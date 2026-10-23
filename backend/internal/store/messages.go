package store

import (
	"database/sql"
	"errors"
	"fmt"
)

// Mensagens prontas do WhatsApp.
//
// O corpo é texto da Maria, com apelidos entre chaves que o frontend troca
// pelos dados do pedido na hora de montar a conversa. O backend não interpreta
// esses apelidos: para ele é texto, e é isso que permite ela inventar uma
// mensagem nova sem precisar de alteração no sistema.

// DefaultMessages são os textos que o ateliê começa usando. Ficam aqui, e não
// na migração, para serem a única fonte: o botão "restaurar" da tela reinsere
// exatamente estes.
func DefaultMessages() []MessageTemplate {
	return []MessageTemplate{
		{
			Name:          "Serviço pronto para retirada",
			SuggestStatus: "Pronto",
			Body: "Oi, {cliente}! Aqui é do Ateliê Delisa.\n\n" +
				"Seu pedido #{pedido} está pronto: {servicos}.\n" +
				"Total: {total}.\n\n" +
				"Pode passar para retirar quando quiser. " +
				"Qualquer dúvida, é só responder por aqui.",
		},
		{
			Name:          "Lembrete da data de entrega",
			SuggestStatus: "Em produção",
			Body: "Oi, {cliente}! Aqui é do Ateliê Delisa.\n\n" +
				"Passando para lembrar do seu pedido #{pedido} ({servicos}), " +
				"combinado para {entrega}.\n\n" +
				"Se precisar mudar o horário, me avise que a gente ajeita.",
		},
		{
			Name:          "Lembrete de pagamento",
			SuggestStatus: "Entregue",
			Body: "Oi, {cliente}! Aqui é do Ateliê Delisa.\n\n" +
				"Sobre o pedido #{pedido}: ficou pendente o valor de {total}.\n\n" +
				"Pode pagar na retirada ou por Pix, como ficar melhor para você.",
		},
	}
}

func (s *Store) ListMessages() ([]MessageTemplate, error) {
	rows, err := s.db.Query(
		`SELECT id, name, body, suggest_status, position
		 FROM message_templates ORDER BY position, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []MessageTemplate{}
	for rows.Next() {
		var m MessageTemplate
		if err := rows.Scan(&m.ID, &m.Name, &m.Body, &m.SuggestStatus, &m.Position); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

func (s *Store) GetMessage(id int64) (MessageTemplate, error) {
	var m MessageTemplate
	err := s.db.QueryRow(
		`SELECT id, name, body, suggest_status, position
		 FROM message_templates WHERE id = ?`, id,
	).Scan(&m.ID, &m.Name, &m.Body, &m.SuggestStatus, &m.Position)
	if errors.Is(err, sql.ErrNoRows) {
		return MessageTemplate{}, ErrNotFound
	}
	if err != nil {
		return MessageTemplate{}, err
	}
	return m, nil
}

func (s *Store) CreateMessage(m MessageTemplate) (MessageTemplate, error) {
	// Sem posição informada, a mensagem nova entra no fim da lista.
	if m.Position == 0 {
		var max int64
		if err := s.db.QueryRow(
			`SELECT COALESCE(MAX(position), 0) FROM message_templates`).Scan(&max); err != nil {
			return MessageTemplate{}, err
		}
		m.Position = max + 1
	}

	res, err := s.db.Exec(
		`INSERT INTO message_templates (name, body, suggest_status, position)
		 VALUES (?, ?, ?, ?)`,
		m.Name, m.Body, m.SuggestStatus, m.Position,
	)
	if err != nil {
		return MessageTemplate{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return MessageTemplate{}, err
	}
	return s.GetMessage(id)
}

func (s *Store) UpdateMessage(id int64, m MessageTemplate) (MessageTemplate, error) {
	res, err := s.db.Exec(
		`UPDATE message_templates SET name = ?, body = ?, suggest_status = ?
		 WHERE id = ?`,
		m.Name, m.Body, m.SuggestStatus, id,
	)
	if err != nil {
		return MessageTemplate{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return MessageTemplate{}, ErrNotFound
	}
	return s.GetMessage(id)
}

func (s *Store) DeleteMessage(id int64) error {
	res, err := s.db.Exec(`DELETE FROM message_templates WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// RestoreMessages joga fora o que houver e devolve os textos padrão. É o
// botão de desistir das edições e começar de novo.
func (s *Store) RestoreMessages() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM message_templates`); err != nil {
		return err
	}
	if err := inserirMensagensPadrao(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// seedMessages insere os textos padrão quando não há nenhuma mensagem
// cadastrada. Roda a cada start, como o seed da tabela de preços: é o que faz
// as mensagens aparecerem no banco que já existia antes desta funcionalidade.
func (s *Store) seedMessages() error {
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM message_templates`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := inserirMensagensPadrao(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func inserirMensagensPadrao(tx *sql.Tx) error {
	for i, m := range DefaultMessages() {
		_, err := tx.Exec(
			`INSERT INTO message_templates (name, body, suggest_status, position)
			 VALUES (?, ?, ?, ?)`,
			m.Name, m.Body, m.SuggestStatus, int64(i+1),
		)
		if err != nil {
			return fmt.Errorf("inserindo a mensagem %q: %w", m.Name, err)
		}
	}
	return nil
}
