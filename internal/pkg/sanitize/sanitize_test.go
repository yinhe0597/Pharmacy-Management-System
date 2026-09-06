package sanitize

import "testing"

func TestMaskIDCard(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"18位身份证", "110101199003074512", "1101**********4512"},
		{"9位保留首尾", "123456789", "1234*6789"},
		{"8位全掩码", "12345678", "********"},
		{"短串全掩码", "123", "***"},
		{"空串", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := MaskIDCard(c.in); got != c.want {
				t.Fatalf("MaskIDCard(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestMaskPhone(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"11位手机号", "13812345678", "138****5678"},
		{"7位保留首尾", "1234567", "1234567"},
		{"6位全掩码", "123456", "******"},
		{"空串", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := MaskPhone(c.in); got != c.want {
				t.Fatalf("MaskPhone(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
