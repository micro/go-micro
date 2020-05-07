package qson

import (
	"fmt"
	"testing"
)

func ExampleUnmarshal() {
	type Ex struct {
		A string `json:"a"`
		B struct {
			C int `json:"c"`
		} `json:"b"`
	}
	var ex Ex
	if err := Unmarshal(&ex, "a=xyz&b[c]=456"); err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", ex)
	// Output: {A:xyz B:{C:456}}
}

type unmarshalT struct {
	A string     `json:"a"`
	B unmarshalB `json:"b"`
}
type unmarshalB struct {
	C int    `json:"c"`
	D string `json:"D"`
}

func TestUnmarshal(t *testing.T) {
	query := "a=xyz&b[c]=456"
	expected := unmarshalT{
		A: "xyz",
		B: unmarshalB{
			C: 456,
		},
	}
	var actual unmarshalT
	err := Unmarshal(&actual, query)
	if err != nil {
		t.Error(err)
	}
	if expected != actual {
		t.Errorf("Expected: %+v Actual: %+v", expected, actual)
	}
}

func ExampleToJSON() {
	b, err := ToJSON("a=xyz&b[c]=456")
	if err != nil {
		panic(err)
	}
	fmt.Printf(string(b))
	// Output: {"a":"xyz","b":{"c":456}}
}

func TestToJSONNested(t *testing.T) {
	query := "bar%5Bone%5D%5Btwo%5D=2&bar[one][red]=112"
	expected := `{"bar":{"one":{"red":112,"two":2}}}`
	actual, err := ToJSON(query)
	if err != nil {
		t.Error(err)
	}
	actualStr := string(actual)
	if actualStr != expected {
		t.Errorf("Expected: %s Actual: %s", expected, actualStr)
	}
}

func TestToJSONPlain(t *testing.T) {
	query := "cat=1&dog=2"
	expected := `{"cat":1,"dog":2}`
	actual, err := ToJSON(query)
	if err != nil {
		t.Error(err)
	}
	actualStr := string(actual)
	if actualStr != expected {
		t.Errorf("Expected: %s Actual: %s", expected, actualStr)
	}
}

func TestToJSONSlice(t *testing.T) {
	query := "cat[]=1&cat[]=34"
	expected := `{"cat":[1,34]}`
	actual, err := ToJSON(query)
	if err != nil {
		t.Error(err)
	}
	actualStr := string(actual)
	if actualStr != expected {
		t.Errorf("Expected: %s Actual: %s", expected, actualStr)
	}
}

func TestToJSONBig(t *testing.T) {
	query := "distinct_id=763_1495187301909_3495&timestamp=1495187523&event=product_add_cart&params%5BproductRefId%5D=8284563078&params%5Bapps%5D%5B%5D=precommend&params%5Bapps%5D%5B%5D=bsales&params%5Bsource%5D=item&params%5Boptions%5D%5Bsegment%5D=cart_recommendation&params%5Boptions%5D%5Btype%5D=up_sell&params%5BtimeExpire%5D=1495187599642&params%5Brecommend_system_product_source%5D=item&params%5Bproduct_id%5D=8284563078&params%5Bvariant_id%5D=27661944134&params%5Bsku%5D=00483332%20(black)&params%5Bsources%5D%5B%5D=product_recommendation&params%5Bcart_token%5D=dc2c336a009edf2762128e65806dfb1d&params%5Bquantity%5D=1&params%5Bnew_popup_upsell_mobile%5D=false&params%5BclientDevice%5D=desktop&params%5BclientIsMobile%5D=false&params%5BclientIsSmallScreen%5D=false&params%5Bnew_popup_crossell_mobile%5D=false&api_key=14c5b7dacea9157029265b174491d340"
	expected := `{"api_key":"14c5b7dacea9157029265b174491d340","distinct_id":"763_1495187301909_3495","event":"product_add_cart","params":{"apps":["precommend","bsales"],"cart_token":"dc2c336a009edf2762128e65806dfb1d","clientDevice":"desktop","clientIsMobile":false,"clientIsSmallScreen":false,"new_popup_crossell_mobile":false,"new_popup_upsell_mobile":false,"options":{"segment":"cart_recommendation","type":"up_sell"},"productRefId":8284563078,"product_id":8284563078,"quantity":1,"recommend_system_product_source":"item","sku":"00483332 (black)","source":"item","sources":["product_recommendation"],"timeExpire":1495187599642,"variant_id":27661944134},"timestamp":1495187523}`
	actual, err := ToJSON(query)
	if err != nil {
		t.Error(err)
	}
	actualStr := string(actual)
	if actualStr != expected {
		t.Errorf("Expected: %s Actual: %s", expected, actualStr)
	}
}

func TestToJSONDuplicateKey(t *testing.T) {
	query := "cat=1&cat=2"
	expected := `{"cat":2}`
	actual, err := ToJSON(query)
	if err != nil {
		t.Error(err)
	}
	actualStr := string(actual)
	if actualStr != expected {
		t.Errorf("Expected: %s Actual: %s", expected, actualStr)
	}
}

func TestSplitKeyAndValue(t *testing.T) {
	param := "a[dog][=cat]=123"
	eKey, eValue := "a[dog][=cat]", "123"
	aKey, aValue, err := splitKeyAndValue(param)
	if err != nil {
		t.Error(err)
	}
	if eKey != aKey {
		t.Errorf("Keys do not match. Expected: %s Actual: %s", eKey, aKey)
	}
	if eValue != aValue {
		t.Errorf("Values do not match. Expected: %s Actual: %s", eValue, aValue)
	}
}

func TestEncodedAmpersand(t *testing.T) {
	query := "a=xyz&b[d]=ben%26jerry"
	expected := unmarshalT{
		A: "xyz",
		B: unmarshalB{
			D: "ben&jerry",
		},
	}
	var actual unmarshalT
	err := Unmarshal(&actual, query)
	if err != nil {
		t.Error(err)
	}
	if expected != actual {
		t.Errorf("Expected: %+v Actual: %+v", expected, actual)
	}
}

func TestEncodedAmpersand2(t *testing.T) {
	query := "filter=parent%3Dflow12345%26request%3Dreq12345&meta.limit=20&meta.offset=0"
	expected := map[string]interface{}{"filter": "parent=flow12345&request=req12345", "meta.limit": float64(20), "meta.offset": float64(0)}
	actual := make(map[string]interface{})
	err := Unmarshal(&actual, query)
	if err != nil {
		t.Error(err)
	}
	for k, v := range actual {
		if nv, ok := expected[k]; !ok || nv != v {
			t.Errorf("Expected: %+v Actual: %+v", expected, actual)
		}
	}
}
