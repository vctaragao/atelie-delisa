package store

import (
	"strings"
	"testing"
)

// TestMensagensPadraoChegamAoBancoQueJaExiste: a tabela nasce vazia e os
// textos padrão são inseridos no start. É o que faz as mensagens aparecerem no
// banco de produção, que já existia muito antes desta tela.
func TestMensagensPadraoChegamAoBancoQueJaExiste(t *testing.T) {
	s := bancoDeTeste(t)

	lista, err := s.ListMessages()
	if err != nil {
		t.Fatalf("listando mensagens: %v", err)
	}
	if len(lista) != len(DefaultMessages()) {
		t.Fatalf("esperava %d mensagens padrão, veio %d", len(DefaultMessages()), len(lista))
	}

	// A mensagem de retirada precisa vir ligada ao status "Pronto": é dela que
	// sai o aviso que a Maria mais manda.
	var pronto *MessageTemplate
	for i := range lista {
		if lista[i].SuggestStatus == "Pronto" {
			pronto = &lista[i]
		}
	}
	if pronto == nil {
		t.Fatalf("nenhuma mensagem sugerida para pedidos prontos: %+v", lista)
	}
	if !strings.Contains(pronto.Body, "{cliente}") || !strings.Contains(pronto.Body, "{pedido}") {
		t.Errorf("a mensagem de retirada perdeu os apelidos: %q", pronto.Body)
	}
}

// TestSeedNaoDuplicaNemRessuscitaMensagemEditada: o seed roda a cada start do
// backend. Se ele reinserisse os padrões, o texto que ela escreveu voltaria a
// conviver com o original a cada reinício do container.
func TestSeedNaoDuplicaNemRessuscitaMensagemEditada(t *testing.T) {
	s := bancoDeTeste(t)

	lista, err := s.ListMessages()
	if err != nil {
		t.Fatalf("listando: %v", err)
	}
	primeira := lista[0]
	primeira.Body = "Texto da Maria, do jeito dela."
	if _, err := s.UpdateMessage(primeira.ID, primeira); err != nil {
		t.Fatalf("editando: %v", err)
	}

	for i := 0; i < 3; i++ {
		if err := s.seedMessages(); err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}

	depois, err := s.ListMessages()
	if err != nil {
		t.Fatalf("listando de novo: %v", err)
	}
	if len(depois) != len(lista) {
		t.Fatalf("o seed duplicou as mensagens: %d viraram %d", len(lista), len(depois))
	}
	if depois[0].Body != "Texto da Maria, do jeito dela." {
		t.Errorf("o texto editado foi sobrescrito: %q", depois[0].Body)
	}
}

func TestMensagemCriaEditaEExclui(t *testing.T) {
	s := bancoDeTeste(t)

	nova, err := s.CreateMessage(MessageTemplate{
		Name:          "Orçamento aprovado?",
		Body:          "Oi, {cliente}! O orçamento ficou em {total}. Posso começar?",
		SuggestStatus: "Orçamento",
	})
	if err != nil {
		t.Fatalf("criando: %v", err)
	}
	if nova.Position == 0 {
		t.Errorf("a mensagem nova deveria entrar no fim da lista, veio posição 0")
	}

	nova.Name = "Orçamento pronto"
	nova.SuggestStatus = ""
	editada, err := s.UpdateMessage(nova.ID, nova)
	if err != nil {
		t.Fatalf("editando: %v", err)
	}
	if editada.Name != "Orçamento pronto" || editada.SuggestStatus != "" {
		t.Errorf("edição não pegou: %+v", editada)
	}
	// A posição não vem no formulário e não pode ser zerada pela edição.
	if editada.Position != nova.Position {
		t.Errorf("a posição mudou de %d para %d", nova.Position, editada.Position)
	}

	if err := s.DeleteMessage(nova.ID); err != nil {
		t.Fatalf("excluindo: %v", err)
	}
	if _, err := s.GetMessage(nova.ID); err != ErrNotFound {
		t.Errorf("esperava ErrNotFound depois de excluir, veio %v", err)
	}
}

// TestRestaurarMensagensVoltaAosPadroes é o botão de desistir das edições.
func TestRestaurarMensagensVoltaAosPadroes(t *testing.T) {
	s := bancoDeTeste(t)

	lista, err := s.ListMessages()
	if err != nil {
		t.Fatalf("listando: %v", err)
	}
	for _, m := range lista {
		if err := s.DeleteMessage(m.ID); err != nil {
			t.Fatalf("limpando: %v", err)
		}
	}
	if _, err := s.CreateMessage(MessageTemplate{Name: "Só minha", Body: "oi"}); err != nil {
		t.Fatalf("criando: %v", err)
	}

	if err := s.RestoreMessages(); err != nil {
		t.Fatalf("restaurando: %v", err)
	}

	depois, err := s.ListMessages()
	if err != nil {
		t.Fatalf("listando depois: %v", err)
	}
	if len(depois) != len(DefaultMessages()) {
		t.Fatalf("esperava %d mensagens, veio %d", len(DefaultMessages()), len(depois))
	}
	for _, m := range depois {
		if m.Name == "Só minha" {
			t.Errorf("a mensagem própria sobreviveu ao restaurar: %+v", m)
		}
	}
}

// TestBackupCarregaAsMensagens: o backup é a única cópia dos dados do ateliê,
// e o texto que ela escreveu é dado dela como qualquer outro. Um arquivo
// antigo, sem a lista, não pode apagar as mensagens de agora.
func TestBackupCarregaAsMensagens(t *testing.T) {
	s := bancoDeTeste(t)

	lista, err := s.ListMessages()
	if err != nil {
		t.Fatalf("listando: %v", err)
	}
	primeira := lista[0]
	primeira.Body = "Texto da Maria."
	if _, err := s.UpdateMessage(primeira.ID, primeira); err != nil {
		t.Fatalf("editando: %v", err)
	}

	backup, err := s.Export()
	if err != nil {
		t.Fatalf("exportando: %v", err)
	}
	if len(backup.Messages) != len(lista) {
		t.Fatalf("o backup levou %d mensagens, esperava %d", len(backup.Messages), len(lista))
	}

	// Mexe nas mensagens depois do backup e restaura: tem de voltar ao texto
	// que estava no arquivo.
	if err := s.DeleteMessage(primeira.ID); err != nil {
		t.Fatalf("excluindo: %v", err)
	}
	if err := s.Import(backup); err != nil {
		t.Fatalf("restaurando: %v", err)
	}
	voltou, err := s.GetMessage(primeira.ID)
	if err != nil {
		t.Fatalf("relendo a mensagem restaurada: %v", err)
	}
	if voltou.Body != "Texto da Maria." {
		t.Errorf("o backup não devolveu o texto: %q", voltou.Body)
	}

	// Arquivo gerado antes desta funcionalidade: sem a lista de mensagens.
	antigo := backup
	antigo.Messages = nil
	if err := s.Import(antigo); err != nil {
		t.Fatalf("restaurando um backup antigo: %v", err)
	}
	aindaLa, err := s.ListMessages()
	if err != nil {
		t.Fatalf("listando: %v", err)
	}
	if len(aindaLa) != len(lista) {
		t.Errorf("o backup antigo apagou as mensagens: sobraram %d de %d", len(aindaLa), len(lista))
	}
}
