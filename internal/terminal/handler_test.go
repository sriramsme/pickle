package terminal

import "testing"

func TestDecodeResize(t *testing.T) {
	tests := []struct {
		name    string
		message string
		wantErr bool
	}{
		{name: "valid", message: `{"type":"resize","cols":120,"rows":40}`},
		{name: "zero columns", message: `{"type":"resize","cols":0,"rows":40}`, wantErr: true},
		{name: "wrong type", message: `{"type":"input","cols":120,"rows":40}`, wantErr: true},
		{name: "malformed", message: `{`, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			size, err := decodeResize([]byte(test.message))
			if test.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}

			if err != nil {
				t.Fatalf("decode resize: %v", err)
			}
			if size.Cols != 120 || size.Rows != 40 {
				t.Fatalf("unexpected size: %dx%d", size.Cols, size.Rows)
			}
		})
	}
}

func TestFindClientSession(t *testing.T) {
	output := []byte("4100\twork\n4200\tapi\n")

	session, ok := findClientSession(output, 4200)
	if !ok {
		t.Fatal("expected to find client session")
	}
	if session != "api" {
		t.Fatalf("unexpected session: %q", session)
	}

	if _, ok := findClientSession(output, 4300); ok {
		t.Fatal("unexpected session for missing client")
	}
}
