package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"atelie/backend/internal/store"
)

// TestParseCentsAceitaOsFormatosDeUmaPlanilha cobre as formas como um preço
// aparece numa planilha feita à mão ou exportada de outro sistema. Errar aqui
// é cobrar dez vezes mais de uma cliente, então cada caso está fixado.
func TestParseCentsAceitaOsFormatosDeUmaPlanilha(t *testing.T) {
	casos := []struct {
		entrada string
		cents   int64
	}{
		{"25", 2500},
		{"25,90", 2590},
		{"25.90", 2590},
		{"R$ 25,90", 2590},
		{"r$25", 2500},
		{"1.250,50", 125050},
		{"1.250", 125000}, // ponto separando milhar, não decimal
		{"1250,5", 125050},
		{"0,99", 99},
		{",50", 50},
		{"", 0},
		{"  35,00  ", 3500},
		{"10,005", 1001}, // terceira casa arredonda, não é truncada
	}
	for _, c := range casos {
		got, err := parseCents(c.entrada)
		if err != nil {
			t.Errorf("parseCents(%q) devolveu erro: %v", c.entrada, err)
			continue
		}
		if got != c.cents {
			t.Errorf("parseCents(%q) = %d, esperava %d", c.entrada, got, c.cents)
		}
	}

	for _, ruim := range []string{"abc", "-5", "25 reais", "R$ x", "12,,5"} {
		if _, err := parseCents(ruim); err == nil {
			t.Errorf("parseCents(%q) deveria recusar", ruim)
		}
	}
}

// TestParseCSVTextEntendeOArquivoDoExcel: o Excel em português salva com ponto
// e vírgula e na codificação do Windows. Sem tratar os dois, os acentos chegam
// trocados e todas as colunas viram uma só.
func TestParseCSVTextEntendeOArquivoDoExcel(t *testing.T) {
	// "nome;categoria\nAlteração;Ajustes" em windows-1252: ç = 0xE7, ã = 0xE3.
	bruto := []byte("nome;categoria\r\nAltera\xe7\xe3o;Ajustes\r\n")

	records, err := parseCSVText(bruto)
	if err != nil {
		t.Fatalf("lendo o CSV: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("esperava 2 linhas, veio %d: %q", len(records), records)
	}
	if len(records[1]) != 2 {
		t.Fatalf("o ponto e vírgula nao separou as colunas: %q", records[1])
	}
	if records[1][0] != "Alteração" {
		t.Errorf("acento perdido: veio %q", records[1][0])
	}

	// Com BOM e vírgula, que é como sai de um sistema que grava UTF-8.
	records, err = parseCSVText([]byte("\xef\xbb\xbfnome,categoria\nAlteração,Ajustes\n"))
	if err != nil {
		t.Fatalf("lendo o CSV em UTF-8: %v", err)
	}
	if records[0][0] != "nome" {
		t.Errorf("o BOM ficou preso no primeiro título: %q", records[0][0])
	}
	if records[1][0] != "Alteração" {
		t.Errorf("esperava Alteração, veio %q", records[1][0])
	}
}

// TestNormalizeKeyReconheceOTituloEscritoDeQualquerJeito garante que o
// cabeçalho da planilha dela case com os campos conhecidos.
func TestNormalizeKeyReconheceOTituloEscritoDeQualquerJeito(t *testing.T) {
	casos := map[string]string{
		"Nome":           "nome",
		" PREÇO ":        "preco",
		"Preço (R$)":     "preco",
		"Observações":    "observacoes",
		"Endereço":       "endereco",
		"Tempo estimado": "tempoestimado",
	}
	for entrada, esperado := range casos {
		if got := normalizeKey(entrada); got != esperado {
			t.Errorf("normalizeKey(%q) = %q, esperava %q", entrada, got, esperado)
		}
	}
}

// ---------- os handlers, sobre um banco de verdade ----------

func novoServidor(t *testing.T) *Server {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "atelie.db"))
	if err != nil {
		t.Fatalf("abrindo o banco de teste: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return New(st)
}

func importar(t *testing.T, s *Server, path, corpo string) (*httptest.ResponseRecorder, importResult) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(corpo))
	rec := httptest.NewRecorder()
	s.Routes([]string{"*"}).ServeHTTP(rec, req)

	var res importResult
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
			t.Fatalf("relatório ilegível: %v — corpo: %s", err, rec.Body.String())
		}
	}
	return rec, res
}

func TestImportClientsCriaEPulaOsQueJaExistem(t *testing.T) {
	s := novoServidor(t)

	csv := "nome;telefone;endereco;observacoes\n" +
		"Maria Souza;(35) 99999-1234;Rua das Flores, 120;Cliente antiga\n" +
		"Joana Lima;(35) 98888-5678;;\n" +
		";(35) 90000-0000;;linha sem nome\n" +
		"maria  souza;(35) 91111-1111;;repetida no proprio arquivo\n"

	rec, res := importar(t, s, "/api/clients/import", csv)
	if rec.Code != http.StatusOK {
		t.Fatalf("esperava 200, veio %d: %s", rec.Code, rec.Body.String())
	}
	if res.Created != 2 {
		t.Errorf("esperava 2 clientes criados, veio %d", res.Created)
	}
	if res.Skipped != 2 {
		t.Errorf("esperava 2 linhas não importadas, veio %d", res.Skipped)
	}
	if len(res.Problems) != 2 {
		t.Errorf("esperava 2 avisos de linha, veio %d: %+v", len(res.Problems), res.Problems)
	}

	clientes, err := s.st.ListClients()
	if err != nil {
		t.Fatal(err)
	}
	if len(clientes) != 2 {
		t.Fatalf("esperava 2 clientes no banco, veio %d", len(clientes))
	}
	var maria store.Client
	for _, c := range clientes {
		if c.Name == "Maria Souza" {
			maria = c
		}
	}
	if maria.ID == 0 {
		t.Fatal("Maria Souza não foi salva")
	}
	if maria.Phone != "(35) 99999-1234" {
		t.Errorf("telefone veio %q", maria.Phone)
	}
	if maria.Address != "Rua das Flores, 120" {
		t.Errorf("o endereço com vírgula dentro das aspas se perdeu: %q", maria.Address)
	}

	// Reimportar o mesmo arquivo não pode duplicar ninguém.
	_, res = importar(t, s, "/api/clients/import", csv)
	if res.Created != 0 {
		t.Errorf("a segunda importação criou %d clientes", res.Created)
	}
	clientes, _ = s.st.ListClients()
	if len(clientes) != 2 {
		t.Errorf("depois de reimportar, o banco tem %d clientes", len(clientes))
	}
}

func TestImportClientsComUpdateCompletaOCadastro(t *testing.T) {
	s := novoServidor(t)
	if _, err := s.st.CreateClient(store.Client{Name: "Maria Souza", Notes: "observação antiga"}); err != nil {
		t.Fatal(err)
	}

	csv := "nome,telefone\nMaria Souza,(35) 99999-1234\n"
	_, res := importar(t, s, "/api/clients/import?update=1", csv)
	if res.Updated != 1 {
		t.Fatalf("esperava 1 atualização, veio %+v", res)
	}

	clientes, _ := s.st.ListClients()
	if len(clientes) != 1 {
		t.Fatalf("esperava 1 cliente, veio %d", len(clientes))
	}
	if clientes[0].Phone != "(35) 99999-1234" {
		t.Errorf("o telefone não foi preenchido: %q", clientes[0].Phone)
	}
	// Coluna ausente no arquivo não apaga o que já estava salvo.
	if clientes[0].Notes != "observação antiga" {
		t.Errorf("a observação foi apagada: %q", clientes[0].Notes)
	}
}

func TestImportServicesLePrecoETempo(t *testing.T) {
	s := novoServidor(t)

	csv := "Nome;Categoria;Preço (R$);Tempo\n" +
		"Barra de cortina;Casa;R$ 45,00;1 h\n" +
		"Ajuste de manga;Ajustes;1.250,50;20 min\n" +
		"Bainha de calça;Ajustes;99,00;30 min\n" + // já vem no cadastro inicial
		"Preço errado;Ajustes;abc;10 min\n"

	rec, res := importar(t, s, "/api/services/import", csv)
	if rec.Code != http.StatusOK {
		t.Fatalf("esperava 200, veio %d: %s", rec.Code, rec.Body.String())
	}
	if res.Created != 2 {
		t.Errorf("esperava 2 serviços criados, veio %d (%+v)", res.Created, res.Problems)
	}
	if res.Skipped != 2 {
		t.Errorf("esperava 2 linhas não importadas, veio %d", res.Skipped)
	}

	servicos, err := s.st.ListServices()
	if err != nil {
		t.Fatal(err)
	}
	porNome := map[string]store.Service{}
	for _, v := range servicos {
		porNome[v.Name] = v
	}
	if v := porNome["Barra de cortina"]; v.PriceCents != 4500 || v.Time != "1 h" {
		t.Errorf("Barra de cortina veio %+v", v)
	}
	if v := porNome["Ajuste de manga"]; v.PriceCents != 125050 {
		t.Errorf("esperava 125050 centavos, veio %d", v.PriceCents)
	}
	// O serviço do cadastro inicial não pode ter sido alterado nem duplicado.
	if v := porNome["Bainha de calça"]; v.PriceCents != 2500 {
		t.Errorf("o preço da bainha mudou sem pedir: %d", v.PriceCents)
	}
	if _, existe := porNome["Preço errado"]; existe {
		t.Error("a linha com preço inválido foi gravada")
	}

	// Com update=1, o mesmo arquivo passa a corrigir o preço.
	_, res = importar(t, s, "/api/services/import?update=1", csv)
	if res.Updated != 3 {
		t.Errorf("esperava 3 atualizações, veio %d", res.Updated)
	}
	servicos, _ = s.st.ListServices()
	for _, v := range servicos {
		if v.Name == "Bainha de calça" && v.PriceCents != 9900 {
			t.Errorf("esperava a bainha a 9900, veio %d", v.PriceCents)
		}
	}
}

// TestImportSemCabecalhoUsaAOrdemDasColunas cobre a planilha montada à mão,
// em que ela digita só os dados.
func TestImportSemCabecalhoUsaAOrdemDasColunas(t *testing.T) {
	s := novoServidor(t)

	_, res := importar(t, s, "/api/clients/import", "Ana Paula;(35) 97777-0000;Rua A, 10\n")
	if res.Created != 1 {
		t.Fatalf("esperava 1 cliente criado, veio %+v", res)
	}
	if len(res.Notes) == 0 {
		t.Error("a tela precisa avisar que leu as colunas pela ordem")
	}
	clientes, _ := s.st.ListClients()
	if clientes[0].Name != "Ana Paula" || clientes[0].Phone != "(35) 97777-0000" {
		t.Errorf("veio %+v", clientes[0])
	}
}

func TestImportRecusaArquivoVazioOuSemNome(t *testing.T) {
	s := novoServidor(t)

	rec, _ := importar(t, s, "/api/clients/import", "")
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("arquivo vazio: esperava 422, veio %d", rec.Code)
	}

	rec, _ = importar(t, s, "/api/clients/import", "telefone;endereco\n(35) 90000-0000;Rua A\n")
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("sem coluna de nome: esperava 422, veio %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "nome") {
		t.Errorf("a mensagem precisa dizer que falta o nome: %s", rec.Body.String())
	}
}
