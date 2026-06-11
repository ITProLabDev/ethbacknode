package abi

import "testing"

func TestParseType_Elementary(t *testing.T) {
	cases := []struct {
		in       string
		wantKind kind
		wantSize int
		wantDyn  bool
		wantCanon string
	}{
		{"uint256", kindUint, 256, false, "uint256"},
		{"uint", kindUint, 256, false, "uint256"},
		{"uint8", kindUint, 8, false, "uint8"},
		{"int256", kindInt, 256, false, "int256"},
		{"int128", kindInt, 128, false, "int128"},
		{"bool", kindBool, 0, false, "bool"},
		{"address", kindAddress, 0, false, "address"},
		{"bytes32", kindFixedBytes, 32, false, "bytes32"},
		{"bytes1", kindFixedBytes, 1, false, "bytes1"},
		{"bytes", kindBytes, 0, true, "bytes"},
		{"string", kindString, 0, true, "string"},
	}
	for _, c := range cases {
		typ, err := parseType(c.in, nil)
		if err != nil {
			t.Fatalf("%s: %v", c.in, err)
		}
		if typ.kind != c.wantKind || typ.size != c.wantSize {
			t.Fatalf("%s: kind=%d size=%d", c.in, typ.kind, typ.size)
		}
		if typ.isDynamic() != c.wantDyn {
			t.Fatalf("%s: isDynamic=%v want %v", c.in, typ.isDynamic(), c.wantDyn)
		}
		if typ.canonical() != c.wantCanon {
			t.Fatalf("%s: canonical=%q want %q", c.in, typ.canonical(), c.wantCanon)
		}
	}
}

func TestParseType_Arrays(t *testing.T) {
	// uint256[] dynamic slice
	sl, err := parseType("uint256[]", nil)
	if err != nil {
		t.Fatal(err)
	}
	if sl.kind != kindSlice || !sl.isDynamic() || sl.elem.kind != kindUint {
		t.Fatalf("uint256[]: %+v", sl)
	}
	if sl.canonical() != "uint256[]" {
		t.Fatalf("canonical=%q", sl.canonical())
	}
	// address[3] fixed array of a static type → static
	fa, err := parseType("address[3]", nil)
	if err != nil {
		t.Fatal(err)
	}
	if fa.kind != kindArray || fa.size != 3 || fa.isDynamic() {
		t.Fatalf("address[3]: %+v dyn=%v", fa, fa.isDynamic())
	}
	if fa.staticSize() != 96 {
		t.Fatalf("address[3] staticSize=%d want 96", fa.staticSize())
	}
	// bytes[2] fixed array of a dynamic type → dynamic
	bd, err := parseType("bytes[2]", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bd.isDynamic() {
		t.Fatal("bytes[2] must be dynamic")
	}
	if bd.canonical() != "bytes[2]" {
		t.Fatalf("canonical=%q", bd.canonical())
	}
}

func TestParseType_Tuple(t *testing.T) {
	comps := []*SmartContractAbiEntryInput{
		{Name: "a", Type: "uint256"},
		{Name: "b", Type: "address"},
	}
	tup, err := parseType("tuple", comps)
	if err != nil {
		t.Fatal(err)
	}
	if tup.kind != kindTuple || len(tup.fields) != 2 || tup.isDynamic() {
		t.Fatalf("tuple: %+v dyn=%v", tup, tup.isDynamic())
	}
	if tup.canonical() != "(uint256,address)" {
		t.Fatalf("canonical=%q", tup.canonical())
	}
	// dynamic tuple
	comps2 := []*SmartContractAbiEntryInput{
		{Name: "a", Type: "uint256"},
		{Name: "s", Type: "string"},
	}
	tup2, err := parseType("tuple", comps2)
	if err != nil {
		t.Fatal(err)
	}
	if !tup2.isDynamic() || tup2.canonical() != "(uint256,string)" {
		t.Fatalf("tuple2: dyn=%v canon=%q", tup2.isDynamic(), tup2.canonical())
	}
	// tuple[] slice of tuples
	ts, err := parseType("tuple[]", comps)
	if err != nil {
		t.Fatal(err)
	}
	if ts.kind != kindSlice || ts.elem.kind != kindTuple || ts.canonical() != "(uint256,address)[]" {
		t.Fatalf("tuple[]: %+v canon=%q", ts, ts.canonical())
	}
}

func TestParseType_Errors(t *testing.T) {
	for _, bad := range []string{"", "uint9", "bytes33", "uint256[", "uint256[0]", "uint256[-1]", "nope"} {
		if _, err := parseType(bad, nil); err == nil {
			t.Fatalf("%q must error", bad)
		}
	}
}
