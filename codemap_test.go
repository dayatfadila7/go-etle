package main

import "testing"

func TestMapViolationCode(t *testing.T) {
	cases := map[string]struct {
		code    string
		want    string
		wantOK  bool
	}{
		"seatbelt":      {code: "1240", want: "PS", wantOK: true},
		"no_helmet":     {code: "1000", want: "PH", wantOK: true},
		"non_helmet":    {code: "151002", want: "PH", wantOK: true},
		"phone":         {code: "1223", want: "HB", wantOK: true},
		"passenger_buckled": {code: "3019", want: "PS", wantOK: true},
		"red_light":     {code: "1625", want: "LM", wantOK: true},
		"ganjil_genap":  {code: "K0002", want: "GG", wantOK: true},
		"E001_stnk":     {code: "E001", want: "ST", wantOK: true},
		"unknown_code":  {code: "9999", want: "", wantOK: false},
		"empty_code":    {code: "", want: "", wantOK: false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, ok := MapViolationCode(tc.code)
			if ok != tc.wantOK {
				t.Errorf("ok = %v, want %v", ok, tc.wantOK)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
