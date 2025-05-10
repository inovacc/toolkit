package cpf

import "testing"

func TestGenerateCPF(t *testing.T) {
	v := GenerateCPF()
	if len(v) != 11 {
		t.Errorf("Invalid CPF: %s", v)
		return
	}

	if !ValidateCPF(v) {
		t.Errorf("Invalid CPF: %s", v)
		return
	}

	v = FormatCPF(v)

	if v[3] != '.' || v[7] != '.' || v[11] != '-' {
		t.Errorf("Incorrectly formatted CPF: %s", v)
		return
	}
}

func TestValidateCPF(t *testing.T) {
	if !ValidateCPF("46216723715") {
		t.Errorf("Invalid CPF: %s", "46216723715")
		return
	}

	if !ValidateCPF("46947778824") {
		t.Errorf("Invalid CPF: %s", "46947778824")
		return
	}

	if !ValidateCPF("16700194113") {
		t.Errorf("Invalid CPF: %s", "16700194113")
		return
	}

	if ValidateCPF("550131936") {
		t.Errorf("Should be invalid CPF: %s", "550131936")
		return
	}
}

func TestFormatCPF(t *testing.T) {
	v := FormatCPF("46216723715")
	if v[3] != '.' || v[7] != '.' || v[11] != '-' {
		t.Errorf("Incorrectly formatted CPF: %s", v)
		return
	}

	v = FormatCPF("46947778824")
	if v[3] != '.' || v[7] != '.' || v[11] != '-' {
		t.Errorf("Incorrectly formatted CPF: %s", v)
		return
	}

	v = FormatCPF("16700194113")
	if v[3] != '.' || v[7] != '.' || v[11] != '-' {
		t.Errorf("Incorrectly formatted CPF: %s", v)
		return
	}
}

func TestOrigin(t *testing.T) {
	v := Origin("46216723715")
	if v != "Rio de Janeiro and Espírito Santo" {
		t.Errorf("Incorrect origin: %s", v)
		return
	}

	v = Origin("46947778824")
	if v != "São Paulo" {
		t.Errorf("Incorrect origin: %s", v)
		return
	}

	v = Origin("16700194113")
	if v != "Distrito Federal, Goiás, Mato Grosso do Sul, and Tocantins" {
		t.Errorf("Incorrect origin: %s", v)
		return
	}

	v = Origin("550131936")
	if v != "" {
		t.Errorf("Incorrect origin: %s", v)
		return
	}
}
