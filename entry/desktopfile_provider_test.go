package entry

import "testing"

func BenchmarkScanMultipleST(b *testing.B) {
	for n := 0; n < b.N; n++ {
		scanMultipleST(candidatesDirectories(), nil)
	}
}

func BenchmarkScanMultipleMT(b *testing.B) {
	for n := 0; n < b.N; n++ {
		scanMultipleMT(candidatesDirectories(), nil)
	}
}

func BenchmarkScanMultiplePL(b *testing.B) {
	for n := 0; n < b.N; n++ {
		scanMultiplePL(candidatesDirectories(), nil)
	}
}

func mapEq(left, right map[string]DesktopFile) bool {
	if len(left) != len(right) {
		return false
	}

	for k, vL := range left {
		vR, ok := right[k]
		if !ok {
			return false
		}
		if vL.Identifier != vR.Identifier {
			return false
		}
		if vL.Name != vR.Name {
			return false
		}
	}
	return true
}

func TestConsistency(t *testing.T) {
	directories := candidatesDirectories()

	res1, err1 := scanMultipleMT(directories, nil)
	if err1 != nil {
		t.Fatal("MT failed")
	}
	if len(res1) != 92 {
		t.Fatal("MT failed")
	}

	res2, err2 := scanMultipleST(directories, nil)
	if err2 != nil {
		t.Fatal("ST failed")
	}
	if len(res2) != 92 {
		t.Fatal("ST failed")
	}

	res3, err3 := scanMultiplePL(directories, nil)
	if err3 != nil {
		t.Fatal("PL failed")
	}
	if len(res3) != 92 {
		t.Fatal(len(res3), "PL failed")
	}

	if !mapEq(res1, res2) {
		t.Fatal("res1 != res2")
	}

	if !mapEq(res2, res3) {
		t.Fatal("res2 != res3")
	}
}
