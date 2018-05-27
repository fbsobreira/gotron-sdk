package base58

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"
)

func TestAlphabetImplStringer(t *testing.T) {
	// interface: Stringer {String()}
	alphabetStr := "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	alphabet := NewAlphabet(alphabetStr)
	if alphabet.String() != alphabetStr {
		t.Errorf("alphabet.String() should be %s, but %s", alphabetStr, alphabet.String())
	}
}

func TestAlphabetFix58Length(t *testing.T) {
	crashed := false
	alphabetStr := "sfdjskdf"

	defer func() {
		if err := recover(); err != nil {
			crashed = true
		}
		if !crashed {
			t.Errorf("NewAlphabet(%s) should crash, but ok", alphabetStr)
		}
	}()

	NewAlphabet(alphabetStr)
}

func TestUnicodeAlphabet(t *testing.T) {
	myAlphabet := NewAlphabet("一二三四五六七八九十壹贰叁肆伍陆柒捌玖零拾佰仟万亿圆甲乙丙丁戊己庚辛壬癸子丑寅卯辰巳午未申酉戌亥金木水火土雷电风雨福")

	testCases := []struct {
		input  []byte
		should string
	}{
		{[]byte{0}, "一"},
		{[]byte{0, 0}, "一一"},
		{[]byte{1}, "二"},
		{[]byte{0, 1}, "一二"},
		{[]byte{1, 1}, "五圆"},
	}

	for _, testItem := range testCases {
		result := Encode(testItem.input, myAlphabet)
		if result != testItem.should {
			t.Errorf("encodeBase58(%v) should %s, but %s", testItem.input, testItem.should, result)
		}

		resultInput, err := Decode(testItem.should, myAlphabet)
		if err != nil {
			t.Errorf("decodeBase58(%s) error : %s", testItem.should, err.Error())
		} else if !bytes.Equal(resultInput, testItem.input) {
			t.Errorf("decodeBase58(%s) should %v, but %v", testItem.should, testItem.input, resultInput)
		}
	}
}

func TestRandCases(t *testing.T) {
	randSeed := time.Now().UnixNano()
	r := rand.New(rand.NewSource(randSeed))

	// generate 256 bytes
	testBytes := make([]byte, r.Intn(1000))
	for idx := range testBytes {
		testBytes[idx] = byte(r.Intn(256))
	}

	alphabet := BitcoinAlphabet
	redix58Bytes, _ := redixTrans256and58(testBytes, 256, 58)
	should58Runes := make([]rune, len(redix58Bytes))
	for idx, num := range redix58Bytes {
		should58Runes[idx] = alphabet.encodeTable[num]
	}

	logTag := fmt.Sprintf("rand[%d]", randSeed)
	// Encode
	calc58Str := Encode(testBytes, alphabet)
	if calc58Str != string(should58Runes) {
		t.Errorf("%s encodeBase58(%v) should %s, but %s", logTag, testBytes, string(should58Runes), calc58Str)
	}

	// Decode
	decodeBytes, err := Decode(string(should58Runes), alphabet)
	if err != nil {
		t.Errorf("%s decodeBase58(%s) error : %s", logTag, string(should58Runes), err.Error())
	} else if !bytes.Equal(decodeBytes, testBytes) {
		t.Errorf("%s decodeBase58(%s) should %v, but %v", logTag, string(should58Runes), testBytes, decodeBytes)
	}
}

// test: [0]: input bytes  [1]: encoded string
var testCases [][][]byte

func init() {
	caseCount := 1000000
	testCases = make([][][]byte, caseCount)
	for i := 0; i < caseCount; i++ {
		data := make([]byte, 32)
		rand.Read(data)
		testCases[i] = [][]byte{data, []byte(Encode(data, BitcoinAlphabet))}
	}
}

func BenchmarkEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Encode([]byte(testCases[i%len(testCases)][0]), BitcoinAlphabet)
	}
}

func BenchmarkDecode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Decode(string(testCases[i%len(testCases)][1]), BitcoinAlphabet)
	}
}

////////////////////////////////////////////////////////////////////////////////
// redix trans
func redixTrans256and58(input []byte, fromRedix uint32, toRedix uint32) ([]byte, error) {
	capacity := int(math.Log(float64(fromRedix))/math.Log(float64(toRedix))) + 1

	zeros := 0
	for zeros < len(input) && input[zeros] == 0 {
		zeros++
	}

	output := make([]byte, 0, capacity)
	for inputPos := zeros; inputPos < len(input); inputPos++ {
		carry := uint32(input[inputPos])
		if carry >= fromRedix {
			return nil, fmt.Errorf("input[%d]=%d invalid for target redix(%d)", inputPos, carry, fromRedix)
		}
		for idx, num := range output {
			carry += fromRedix * uint32(num)
			output[idx] = byte(carry % uint32(toRedix))
			carry /= toRedix
		}

		for carry != 0 {
			output = append(output, byte(carry%toRedix))
			carry /= toRedix
		}

	}

	for i := 0; i < zeros; i++ {
		output = append(output, 0)
	}

	// reverse
	for idx := 0; idx < len(output)/2; idx++ {
		output[len(output)-idx-1], output[idx] = output[idx], output[len(output)-idx-1]
	}
	return output, nil
}

func TestTransRedix(t *testing.T) {
	for _, testItem := range []struct {
		num256 []byte
		num58  []byte
	}{
		{[]byte{0}, []byte{0}},
		{[]byte{0, 0}, []byte{0, 0}},
		{[]byte{0, 0, 0}, []byte{0, 0, 0}},
		{[]byte{1}, []byte{1}},
		{[]byte{57}, []byte{57}},
		{[]byte{58}, []byte{1, 0}},
		{[]byte{1, 0}, []byte{4, 24}},
		{[]byte{14, 239}, []byte{1, 7, 53}},
	} {
		calc58, err := redixTrans256and58(testItem.num256, 256, 58)
		if err != nil {
			t.Errorf("redix256to58: %v should be %v, but error %s", testItem.num256, testItem.num58, err.Error())
		} else if !bytes.Equal(calc58, testItem.num58) {
			t.Errorf("redix256to58: %v should be %v, but %v", testItem.num256, testItem.num58, calc58)
		}

		calc256, err := redixTrans256and58(testItem.num58, 58, 256)
		if err != nil {
			t.Errorf("redix58to256: %v should be %v, but error %s", testItem.num58, testItem.num256, err.Error())
		} else if !bytes.Equal(calc256, testItem.num256) {
			t.Errorf("redix58to256: %v should be %v, but %v", testItem.num58, testItem.num256, calc256)
		}
	}
}
