package queue

import "testing"

func TestMatchPhraseTriggerWithTermAccentsAndSemicolons(t *testing.T) {
	phrase, ok := matchPhraseTriggerWithTerm("reunião agendada, horário marcado; pix 1.500,00", "Perfeito! Sua reuniao agendada para amanhã às 15h.")
	if !ok || phrase != "reunião agendada" {
		t.Fatalf("expected accent-insensitive match for 'reunião agendada', got ok=%v phrase=%q", ok, phrase)
	}

	phraseDec, okDec := matchPhraseTriggerWithTerm("pix 1.500,00, confirmado", "Recebemos o pix 1.500,00 agora!")
	if !okDec || phraseDec != "pix 1.500,00" {
		t.Fatalf("expected decimal comma preserved inside trigger term, got ok=%v phrase=%q", okDec, phraseDec)
	}
}
