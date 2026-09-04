package register

import "testing"

const pago = "Domiciliación bancaria"

func TestSheetColumnsPerModality(t *testing.T) {
	tests := []struct {
		modality string
		group    string
		amount   string
		obs      string
	}{
		{"40€/mes (2 temas/mes)", "2 temas/mes", "40 EUR/mes", "Forma de pago: Domiciliación bancaria"},
		{"60€/mes (3 temas/mes)", "3 temas/mes", "60 EUR/mes", "Forma de pago: Domiciliación bancaria"},
		{"80€/mes (4 temas/mes)", "4 temas/mes", "80 EUR/mes", "Forma de pago: Domiciliación bancaria"},
		{"100€/mes (5 temas/mes)", "5 temas/mes", "100 EUR/mes", "Forma de pago: Domiciliación bancaria"},
		{"120€/mes (6 temas/mes)", "6 temas/mes", "120 EUR/mes", "Forma de pago: Domiciliación bancaria"},
		{"ministerios-pago-unico-360", "", "360 EUR", "Pago único 360 EUR | Forma de pago: Domiciliación bancaria"},
	}

	for _, tc := range tests {
		t.Run(tc.modality, func(t *testing.T) {
			if got := extractGroup(tc.modality); got != tc.group {
				t.Errorf("GRUPO = %q, want %q", got, tc.group)
			}
			if got := extractAmount(tc.modality); got != tc.amount {
				t.Errorf("IMPORTE/MES = %q, want %q", got, tc.amount)
			}
			if got := buildObservaciones(pago, tc.modality); got != tc.obs {
				t.Errorf("OBSERVACIONES = %q, want %q", got, tc.obs)
			}
			if !ValidModality(tc.modality) {
				t.Errorf("ValidModality(%q) = false, want true", tc.modality)
			}
		})
	}
}

func TestValidModalityRejectsUnknown(t *testing.T) {
	for _, m := range []string{"hacked", "", "MINISTERIOS- PAGO UNICO- 360EUR"} {
		if ValidModality(m) {
			t.Errorf("ValidModality(%q) = true, want false", m)
		}
	}
}

func TestModalityLabelFallsBackToInput(t *testing.T) {
	if got := modalityLabel("unknown"); got != "unknown" {
		t.Errorf("modalityLabel(unknown) = %q, want %q", got, "unknown")
	}
	want := "MINISTERIOS - PAGO ÚNICO - 360 €"
	if got := modalityLabel("ministerios-pago-unico-360"); got != want {
		t.Errorf("modalityLabel = %q, want %q", got, want)
	}
}
