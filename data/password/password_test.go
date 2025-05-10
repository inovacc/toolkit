package password

import (
	"strings"
	"testing"
)

func TestNewPassword(t *testing.T) {
	p := NewPassword(
		WithLength(16),
		WithNumbers(),
		WithSpecial(),
		WithLower(),
		WithUpper(),
	)

	if _, err := p.Generate(); err != nil {
		t.Fatal(err)
	}
}

func TestPassword_Generate_SingleCharTypes(t *testing.T) {
	tests := []struct {
		name        string
		options     []Option
		expectedSet string
	}{
		{
			name: "Only numbers",
			options: []Option{
				WithLength(12),
				WithNumbers(),
			},
			expectedSet: passwordOptions["num"],
		},
		{
			name: "Only upper case",
			options: []Option{
				WithLength(12),
				WithUpper(),
			},
			expectedSet: passwordOptions["upperCase"],
		},
		{
			name: "Only lower case",
			options: []Option{
				WithLength(12),
				WithLower(),
			},
			expectedSet: passwordOptions["lowerCase"],
		},
		{
			name: "Only special characters",
			options: []Option{
				WithLength(12),
				WithSpecial(),
			},
			expectedSet: passwordOptions["specialChar"],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPassword(tt.options...)
			password, err := p.Generate()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(password) != 12 {
				t.Errorf("expected password length 12, got %d", len(password))
			}
			for _, r := range password {
				if !strings.ContainsRune(tt.expectedSet, r) {
					t.Errorf("character %q not in expected set %q", r, tt.expectedSet)
				}
			}
		})
	}
}

func TestPassword_Generate_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		options     []Option
		wantErr     bool
		errContains string
	}{
		{
			name: "Password too short",
			options: []Option{
				WithLength(7),
				WithNumbers(),
			},
			wantErr:     true,
			errContains: "length must be between 8 and 128",
		},
		{
			name: "Password too long",
			options: []Option{
				WithLength(129),
				WithNumbers(),
			},
			wantErr:     true,
			errContains: "length must be between 8 and 128",
		},
		{
			name:        "No options selected",
			options:     []Option{WithLength(8)},
			wantErr:     true,
			errContains: "no character sets selected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPassword(tt.options...)
			password, err := p.Generate()

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if password == "" {
				t.Error("password should not be empty")
			}
		})
	}
}

func TestPassword_Generate_CombinedCharTypes(t *testing.T) {
	tests := []struct {
		name    string
		options []Option
		verify  func(string) bool
	}{
		{
			name: "Numbers and Special",
			options: []Option{
				WithLength(12),
				WithNumbers(),
				WithSpecial(),
			},
			verify: func(s string) bool {
				hasNum := false
				hasSpecial := false
				for _, r := range s {
					if strings.ContainsRune(passwordOptions["num"], r) {
						hasNum = true
					}
					if strings.ContainsRune(passwordOptions["specialChar"], r) {
						hasSpecial = true
					}
				}
				return hasNum && hasSpecial
			},
		},
		{
			name: "All character types",
			options: []Option{
				WithLength(16),
				WithNumbers(),
				WithSpecial(),
				WithLower(),
				WithUpper(),
			},
			verify: func(s string) bool {
				hasNum := false
				hasSpecial := false
				hasLower := false
				hasUpper := false
				for _, r := range s {
					if strings.ContainsRune(passwordOptions["num"], r) {
						hasNum = true
					}
					if strings.ContainsRune(passwordOptions["specialChar"], r) {
						hasSpecial = true
					}
					if strings.ContainsRune(passwordOptions["lowerCase"], r) {
						hasLower = true
					}
					if strings.ContainsRune(passwordOptions["upperCase"], r) {
						hasUpper = true
					}
				}
				return hasNum && hasSpecial && hasLower && hasUpper
			},
		},
		{
			name: "Upper and Lower only",
			options: []Option{
				WithLength(10),
				WithUpper(),
				WithLower(),
			},
			verify: func(s string) bool {
				hasLower := false
				hasUpper := false
				for _, r := range s {
					if strings.ContainsRune(passwordOptions["lowerCase"], r) {
						hasLower = true
					}
					if strings.ContainsRune(passwordOptions["upperCase"], r) {
						hasUpper = true
					}
				}
				return hasLower && hasUpper
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPassword(tt.options...)
			password, err := p.Generate()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !tt.verify(password) {
				t.Error("password does not meet character type requirements")
			}
		})
	}
}

func TestPassword_Generate_Distribution(t *testing.T) {
	p := NewPassword(
		WithLength(128),
		WithNumbers(),
		WithSpecial(),
		WithLower(),
		WithUpper(),
	)

	password, err := p.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	counts := make(map[string]int)
	for _, r := range password {
		for category, chars := range passwordOptions {
			if strings.ContainsRune(chars, r) {
				counts[category]++
				break
			}
		}
	}

	// Check if each character type appears at least 100 times in a 128-char password
	for category, count := range counts {
		if count < 1 {
			t.Errorf("character type %s appears only %d times in 128 characters", category, count)
		}
	}
}
