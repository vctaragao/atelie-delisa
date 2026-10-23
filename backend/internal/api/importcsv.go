package api

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"atelie/backend/internal/store"
)

// Importação de clientes e da tabela de preços a partir de um arquivo CSV.
//
// O corpo da requisição é o arquivo cru (text/csv), e não JSON: ler os bytes
// como vieram é o que permite recuperar os acentos de uma planilha salva pelo
// Excel, que não grava em UTF-8.
//
// A importação não é atômica de propósito. Cada linha é tratada por conta
// própria e as que falharem voltam no relatório com o número da linha. Numa
// lista de cinquenta nomes, é mais útil gravar quarenta e oito e dizer quais
// duas precisam de correção do que recusar o arquivo inteiro.

// maxProblems limita o relatório. Um arquivo todo errado não precisa devolver
// uma mensagem por linha para a tela ficar compreensível.
const maxProblems = 100

// importResult é o relatório mostrado na tela depois da importação.
type importResult struct {
	Created  int           `json:"created"`
	Updated  int           `json:"updated"`
	Skipped  int           `json:"skipped"`
	Notes    []string      `json:"notes"`
	Problems []importIssue `json:"problems"`
	Omitted  int           `json:"omitted"`
}

type importIssue struct {
	Line    int    `json:"line"`
	Message string `json:"message"`
}

func (res *importResult) note(line int, msg string) {
	if len(res.Problems) >= maxProblems {
		res.Omitted++
		return
	}
	res.Problems = append(res.Problems, importIssue{Line: line, Message: msg})
}

// importField descreve um campo do CSV: o nome usado aqui no código e as
// formas como ele pode aparecer no cabeçalho da planilha dela.
type importField struct {
	name    string
	aliases []string
}

var clientFields = []importField{
	{"nome", []string{"nome", "cliente", "nomedocliente", "nomecompleto"}},
	{"telefone", []string{"telefone", "celular", "whatsapp", "fone", "tel", "contato"}},
	{"endereco", []string{"endereco", "rua", "enderecocompleto"}},
	{"observacoes", []string{"observacoes", "observacao", "obs", "notas", "anotacoes"}},
}

var serviceFields = []importField{
	{"nome", []string{"nome", "servico", "nomedoservico", "descricao"}},
	{"categoria", []string{"categoria", "grupo", "tipo"}},
	{"preco", []string{"preco", "precopadrao", "precounitario", "valor"}},
	{"tempo", []string{"tempo", "tempoestimado", "duracao", "prazo"}},
}

// ---------- leitura do arquivo ----------

// parseCSVText interpreta os bytes do arquivo como CSV, resolvendo sozinho a
// codificação e o separador — a planilha chega do jeito que o Excel salvou.
func parseCSVText(raw []byte) ([][]string, error) {
	text := decodeText(raw)
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("o arquivo está vazio")
	}

	rd := csv.NewReader(strings.NewReader(text))
	rd.Comma = detectDelimiter(text)
	rd.FieldsPerRecord = -1 // linhas com menos colunas são normais aqui
	rd.LazyQuotes = true    // aspas soltas no meio do texto não derrubam a leitura
	rd.TrimLeadingSpace = true

	records, err := rd.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("não foi possível ler o arquivo como CSV: %w", err)
	}
	return records, nil
}

// decodeText devolve o conteúdo em UTF-8. O Excel em português salva CSV na
// codificação do Windows; sem esta conversão, "Alteração" chegaria com os
// caracteres trocados.
func decodeText(b []byte) string {
	b = bytes.TrimPrefix(b, []byte{0xEF, 0xBB, 0xBF}) // BOM que o Excel escreve
	if utf8.Valid(b) {
		return string(b)
	}
	// A faixa dos acentos é igual no windows-1252 e no latin-1, e é só ela
	// que aparece em nome de cliente ou de serviço.
	var sb strings.Builder
	sb.Grow(len(b))
	for _, c := range b {
		sb.WriteRune(rune(c))
	}
	return sb.String()
}

// detectDelimiter escolhe o separador contando a primeira linha. O Excel
// configurado em português grava com ponto e vírgula, não com vírgula.
func detectDelimiter(text string) rune {
	line := text
	if i := strings.IndexAny(text, "\r\n"); i >= 0 {
		line = text[:i]
	}
	best, bestN := ',', strings.Count(line, ",")
	for _, c := range []rune{';', '\t'} {
		if n := strings.Count(line, string(c)); n > bestN {
			best, bestN = c, n
		}
	}
	return best
}

// csvTable é o arquivo já interpretado: em que coluna está cada campo e as
// linhas de dados, cada uma sabendo seu número no arquivo.
type csvTable struct {
	cols  map[string]int
	rows  []csvRow
	notes []string
}

type csvRow struct {
	line   int
	fields []string
}

func (t csvTable) get(row csvRow, field string) string {
	i, ok := t.cols[field]
	if !ok || i < 0 || i >= len(row.fields) {
		return ""
	}
	return strings.TrimSpace(row.fields[i])
}

// buildTable casa o cabeçalho com os campos conhecidos. Sem cabeçalho
// reconhecível, as colunas valem pela ordem — uma planilha montada à mão
// raramente tem a primeira linha com os títulos.
func buildTable(records [][]string, fields []importField) csvTable {
	t := csvTable{cols: map[string]int{}}

	start := 0
	if len(records) > 0 && looksLikeHeader(records[0], fields) {
		for i, cell := range records[0] {
			key := normalizeKey(cell)
			for _, f := range fields {
				if _, taken := t.cols[f.name]; taken {
					continue
				}
				for _, a := range f.aliases {
					if key == a {
						t.cols[f.name] = i
						break
					}
				}
			}
		}
		start = 1
	} else {
		order := make([]string, 0, len(fields))
		for i, f := range fields {
			t.cols[f.name] = i
			order = append(order, f.name)
		}
		t.notes = append(t.notes,
			"O arquivo não tem uma primeira linha com os títulos das colunas, "+
				"então elas foram lidas nesta ordem: "+strings.Join(order, ", ")+".")
	}

	for i := start; i < len(records); i++ {
		if blankRow(records[i]) {
			continue
		}
		t.rows = append(t.rows, csvRow{line: i + 1, fields: records[i]})
	}
	return t
}

func looksLikeHeader(row []string, fields []importField) bool {
	for _, cell := range row {
		key := normalizeKey(cell)
		for _, f := range fields {
			for _, a := range f.aliases {
				if key == a {
					return true
				}
			}
		}
	}
	return false
}

func blankRow(row []string) bool {
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

var semAcento = strings.NewReplacer(
	"á", "a", "à", "a", "ã", "a", "â", "a", "ä", "a",
	"é", "e", "ê", "e", "è", "e",
	"í", "i", "î", "i", "ì", "i",
	"ó", "o", "õ", "o", "ô", "o", "ö", "o",
	"ú", "u", "û", "u", "ù", "u", "ü", "u",
	"ç", "c",
)

// normalizeKey reduz o título da coluna a letras e números sem acento, para
// que "Preço (R$)", "preco" e "PREÇO" cheguem todos a "preco".
func normalizeKey(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if i := strings.IndexByte(s, '('); i > 0 {
		s = s[:i]
	}
	s = semAcento.Replace(s)
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// nameKey é a chave de comparação de nomes: é ela que decide se a linha do
// arquivo é alguém que já está cadastrado. Ignora caixa, acento e espaço
// repetido, porque "Ana Paula" e "ana  paula" são a mesma pessoa.
func nameKey(s string) string {
	s = semAcento.Replace(strings.ToLower(strings.TrimSpace(s)))
	return strings.Join(strings.Fields(s), " ")
}

// ---------- preço ----------

// parseCents converte um preço escrito à mão em centavos. Aceita "25",
// "25,90", "R$ 1.250,50" e "25.90", que é como os valores aparecem nas
// planilhas e nos sistemas de onde ela pode exportar.
func parseCents(raw string) (int64, error) {
	invalid := func() (int64, error) {
		return 0, fmt.Errorf("preço inválido: %q", strings.TrimSpace(raw))
	}

	s := strings.TrimSpace(raw)
	for _, junk := range []string{" ", " ", "R$", "r$", "$"} {
		s = strings.ReplaceAll(s, junk, "")
	}
	if s == "" {
		return 0, nil
	}
	if strings.HasPrefix(s, "-") {
		return 0, fmt.Errorf("preço negativo: %q", strings.TrimSpace(raw))
	}

	lastComma := strings.LastIndex(s, ",")
	lastDot := strings.LastIndex(s, ".")

	// Separador decimal: a vírgula sempre que vier depois do ponto
	// ("1.250,50"); o ponto só quando não estiver separando milhar —
	// "25.90" é um preço, "1.250" é mil duzentos e cinquenta.
	dec := -1
	switch {
	case lastComma > lastDot:
		dec = lastComma
	case lastDot > lastComma && len(s)-lastDot-1 != 3:
		dec = lastDot
	}

	intPart, fracPart := s, ""
	if dec >= 0 {
		intPart, fracPart = s[:dec], s[dec+1:]
	}

	onlyDigits := func(v string) (string, bool) {
		for _, r := range v {
			if r < '0' || r > '9' {
				return "", false
			}
		}
		return v, true
	}

	// Na parte inteira, o que sobrou de ponto ou vírgula só pode ser separador
	// de milhar — e separador de milhar vem sempre com três dígitos atrás
	// dele. É isso que separa "1.250" de um "12,,5" digitado errado.
	grupos := strings.FieldsFunc(intPart, func(r rune) bool { return r == '.' || r == ',' })
	if strings.Count(intPart, ".")+strings.Count(intPart, ",") != len(grupos)-1 && intPart != "" {
		return invalid()
	}
	for i, g := range grupos {
		if _, ok := onlyDigits(g); !ok {
			return invalid()
		}
		if i > 0 && len(g) != 3 {
			return invalid()
		}
		if i == 0 && len(grupos) > 1 && (g == "" || len(g) > 3) {
			return invalid()
		}
	}
	ip := strings.Join(grupos, "")

	fp, ok := onlyDigits(fracPart)
	if !ok {
		return invalid()
	}
	if ip == "" && fp == "" {
		return invalid()
	}
	if len(ip) > 12 {
		return 0, fmt.Errorf("preço fora da faixa aceita: %q", strings.TrimSpace(raw))
	}

	var reais int64
	if ip != "" {
		n, err := strconv.ParseInt(ip, 10, 64)
		if err != nil {
			return invalid()
		}
		reais = n
	}

	var cents int64
	switch {
	case len(fp) == 1:
		cents = int64(fp[0]-'0') * 10
	case len(fp) >= 2:
		n, err := strconv.ParseInt(fp[:2], 10, 64)
		if err != nil {
			return invalid()
		}
		cents = n
		if len(fp) > 2 && fp[2] >= '5' {
			cents++ // arredonda a terceira casa, em vez de truncá-la
		}
	}
	return reais*100 + cents, nil
}

// ---------- handlers ----------

// readImportCSV lê o corpo da requisição e devolve a tabela interpretada.
func readImportCSV(w http.ResponseWriter, r *http.Request, fields []importField) (csvTable, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Não foi possível ler o arquivo enviado.")
		return csvTable{}, false
	}

	records, err := parseCSVText(raw)
	if err != nil {
		writeErr(w, http.StatusUnprocessableEntity, capitalize(err.Error())+".")
		return csvTable{}, false
	}

	t := buildTable(records, fields)
	if _, ok := t.cols["nome"]; !ok {
		writeErr(w, http.StatusUnprocessableEntity,
			"O arquivo precisa de uma coluna com o nome. Verifique a primeira linha da planilha.")
		return csvTable{}, false
	}
	if len(t.rows) == 0 {
		writeErr(w, http.StatusUnprocessableEntity, "O arquivo não tem nenhuma linha com dados.")
		return csvTable{}, false
	}
	return t, true
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	return string(unicode.ToUpper(r[0])) + string(r[1:])
}

// wantsUpdate diz se as linhas que já existem devem ser atualizadas. O padrão
// é não mexer no que já está cadastrado.
func wantsUpdate(r *http.Request) bool {
	return r.URL.Query().Get("update") == "1"
}

func (s *Server) importClients(w http.ResponseWriter, r *http.Request) {
	t, ok := readImportCSV(w, r, clientFields)
	if !ok {
		return
	}
	update := wantsUpdate(r)

	current, err := s.st.ListClients()
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	byName := map[string]store.Client{}
	for _, c := range current {
		byName[nameKey(c.Name)] = c
	}

	res := importResult{Notes: t.notes, Problems: []importIssue{}}
	for _, row := range t.rows {
		in := store.Client{
			Name:    t.get(row, "nome"),
			Phone:   t.get(row, "telefone"),
			Address: t.get(row, "endereco"),
			Notes:   t.get(row, "observacoes"),
		}
		if in.Name == "" {
			res.Skipped++
			res.note(row.line, "linha sem nome — não importada")
			continue
		}

		key := nameKey(in.Name)
		if existing, dup := byName[key]; dup {
			if !update {
				res.Skipped++
				res.note(row.line, fmt.Sprintf("%q já estava cadastrado — nada foi alterado", existing.Name))
				continue
			}
			// Campo vazio no arquivo não apaga o que já está salvo.
			merged := existing
			if in.Phone != "" {
				merged.Phone = in.Phone
			}
			if in.Address != "" {
				merged.Address = in.Address
			}
			if in.Notes != "" {
				merged.Notes = in.Notes
			}
			saved, err := s.st.UpdateClient(existing.ID, merged)
			if err != nil {
				res.note(row.line, fmt.Sprintf("não foi possível atualizar %q", in.Name))
				continue
			}
			byName[key] = saved
			res.Updated++
			continue
		}

		saved, err := s.st.CreateClient(in)
		if err != nil {
			res.note(row.line, fmt.Sprintf("não foi possível salvar %q", in.Name))
			continue
		}
		byName[key] = saved
		res.Created++
	}

	writeJSON(w, http.StatusOK, res)
}

func (s *Server) importServices(w http.ResponseWriter, r *http.Request) {
	t, ok := readImportCSV(w, r, serviceFields)
	if !ok {
		return
	}
	update := wantsUpdate(r)

	current, err := s.st.ListServices()
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	byName := map[string]store.Service{}
	for _, v := range current {
		byName[nameKey(v.Name)] = v
	}

	res := importResult{Notes: t.notes, Problems: []importIssue{}}
	if _, ok := t.cols["preco"]; !ok {
		res.Notes = append(res.Notes,
			"O arquivo não tem coluna de preço: os serviços novos entraram com R$ 0,00.")
	}

	for _, row := range t.rows {
		name := t.get(row, "nome")
		if name == "" {
			res.Skipped++
			res.note(row.line, "linha sem nome de serviço — não importada")
			continue
		}
		priceRaw := t.get(row, "preco")
		cents, err := parseCents(priceRaw)
		if err != nil {
			res.Skipped++
			res.note(row.line, fmt.Sprintf("%s em %q — não importada", err.Error(), name))
			continue
		}

		in := store.Service{
			Name:       name,
			Category:   t.get(row, "categoria"),
			PriceCents: cents,
			Time:       t.get(row, "tempo"),
		}

		key := nameKey(name)
		if existing, dup := byName[key]; dup {
			if !update {
				res.Skipped++
				res.note(row.line, fmt.Sprintf("%q já estava na tabela de preços — nada foi alterado", existing.Name))
				continue
			}
			merged := existing
			if priceRaw != "" {
				merged.PriceCents = cents
			}
			if in.Category != "" {
				merged.Category = in.Category
			}
			if in.Time != "" {
				merged.Time = in.Time
			}
			saved, err := s.st.UpdateService(existing.ID, merged)
			if err != nil {
				res.note(row.line, fmt.Sprintf("não foi possível atualizar %q", name))
				continue
			}
			byName[key] = saved
			res.Updated++
			continue
		}

		saved, err := s.st.CreateService(in)
		if err != nil {
			res.note(row.line, fmt.Sprintf("não foi possível salvar %q", name))
			continue
		}
		byName[key] = saved
		res.Created++
	}

	writeJSON(w, http.StatusOK, res)
}
