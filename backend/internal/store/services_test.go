package store

import "testing"

// TestServicoNovoEntraEmOrdemAlfabetica: a tabela de preços e o seletor do
// pedido seguem a ordem desta lista. Antes a ordenação era por categoria e
// depois por nome, então um serviço novo sem categoria ia para o topo da
// tabela, longe de onde ela procurava pelo nome.
func TestServicoNovoEntraEmOrdemAlfabetica(t *testing.T) {
	s := bancoDeTeste(t)

	// Os dois entram sem categoria, como ela cadastra na correria, e um deles
	// com a primeira letra minúscula.
	novos := []Service{
		{Name: "Casa de botão", PriceCents: 800},
		{Name: "aplicação de renda", PriceCents: 1500},
	}
	for _, v := range novos {
		if _, err := s.CreateService(v); err != nil {
			t.Fatalf("cadastrando %q: %v", v.Name, err)
		}
	}

	lista, err := s.ListServices()
	if err != nil {
		t.Fatalf("listando serviços: %v", err)
	}

	// Os seis primeiros são o cadastro inicial, com os dois novos no lugar
	// alfabético que lhes cabe — e não no fim nem no começo da lista.
	esperado := []string{
		"Ajuste de cintura",
		"Ajuste lateral",
		"aplicação de renda",
		"Bainha de calça",
		"Bainha de vestido",
		"Casa de botão",
		"Reparo simples",
		"Troca de zíper",
	}
	if len(lista) != len(esperado) {
		t.Fatalf("esperava %d serviços, veio %d", len(esperado), len(lista))
	}
	for i, nome := range esperado {
		if lista[i].Name != nome {
			nomes := make([]string, len(lista))
			for j, v := range lista {
				nomes[j] = v.Name
			}
			t.Fatalf("na posição %d esperava %q; a lista veio %v", i, nome, nomes)
		}
	}
}
