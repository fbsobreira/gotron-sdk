package abi

import (
	"math/big"
	"testing"
)

func TestABIParam(t *testing.T) {
	//b, err := GetPaddedParam([]Param{{"address[2]": []string{"TEvHMZWyfjCAdDJEKYxYVL8rRpigddLC1R", "TEvHMZWyfjCAdDJEKYxYVL8rRpigddLC1R"}}})
	b, err := GetPaddedParam([]Param{
		{"string": "KLV Test Token"},
		{"string": "KLV"},
		{"uint8": uint8(6)},
		{"uint256": big.NewInt(1000000000)},
	})
	if err != nil {
		t.Errorf(" %+v", err)
	}
	if len(b) != 256 {
		t.Errorf("Wrong length %d/%d", len(b), 256)
	}
}

func TestABIParamArray(t *testing.T) {
	b, err := GetPaddedParam([]Param{{"address[2]": []string{"TEvHMZWyfjCAdDJEKYxYVL8rRpigddLC1R", "TEvHMZWyfjCAdDJEKYxYVL8rRpigddLC1R"}}})
	if err != nil {
		t.Errorf(" %+v", err)
	}
	if len(b) != 64 {
		t.Errorf("Wrong length %d/%d", len(b), 64)
	}

}
