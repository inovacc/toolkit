package cnpj

import "testing"

func TestGenerateCNPJ(t *testing.T) {
	v := GenerateCNPJ()
	if len(v) != 14 {
		t.Errorf("Invalid CNPJ: %s", v)
		return
	}

	if !ValidateCNPJ(v) {
		t.Errorf("Invalid CNPJ: %s", v)
		return
	}

	v = FormatCNPJ(v)

	if v[2] != '.' || v[6] != '.' || v[10] != '/' || v[15] != '-' {
		t.Errorf("Incorrectly formatted CNPJ: %s", v)
		return
	}
}

func TestValidateCNPJ(t *testing.T) {
	if !ValidateCNPJ("OTWXQENJDKC620") {
		t.Errorf("Invalid CNPJ: %s", "OTWXQENJDKC620")
		return
	}

	if !ValidateCNPJ("RZYYOMTNOLSV26") {
		t.Errorf("Invalid CNPJ: %s", "RZYYOMTNOLSV26")
		return
	}

	if !ValidateCNPJ("D6RJ1CUTQQAA22") {
		t.Errorf("Invalid CNPJ: %s", "D6RJ1CUTQQAA22")
		return
	}

	if ValidateCNPJ("INVALIDCNPJ") {
		t.Errorf("Should be invalid CNPJ: %s", "INVALIDCNPJ")
		return
	}
}

func TestFormatCNPJ(t *testing.T) {
	v := FormatCNPJ("OTWXQENJDKC620")
	if v[2] != '.' || v[6] != '.' || v[10] != '/' || v[15] != '-' {
		t.Errorf("Incorrectly formatted CNPJ: %s", v)
		return
	}

	v = FormatCNPJ("RZYYOMTNOLSV26")
	if v[2] != '.' || v[6] != '.' || v[10] != '/' || v[15] != '-' {
		t.Errorf("Incorrectly formatted CNPJ: %s", v)
		return
	}

	v = FormatCNPJ("D6RJ1CUTQQAA22")
	if v[2] != '.' || v[6] != '.' || v[10] != '/' || v[15] != '-' {
		t.Errorf("Incorrectly formatted CNPJ: %s", v)
		return
	}
}
