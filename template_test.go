package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"text/template"
)

type templateTestList []struct {
	tmpl     string
	context  interface{}
	expected string
}

func (tests templateTestList) run(t *testing.T, prefix string) {
	for n, test := range tests {
		tmplName := fmt.Sprintf("%s-test-%d", prefix, n)
		tmpl := template.Must(newTemplate(tmplName).Parse(test.tmpl))

		var b bytes.Buffer
		err := tmpl.ExecuteTemplate(&b, tmplName, test.context)
		if err != nil {
			t.Fatalf("Error executing template: %v (test %s)", err, tmplName)
		}

		got := b.String()
		if test.expected != got {
			t.Fatalf("Incorrect output found; expected %s, got %s (test %s)", test.expected, got, tmplName)
		}
	}
}

func TestContains(t *testing.T) {
	env := map[string]string{
		"PORT": "1234",
	}

	if !contains(env, "PORT") {
		t.Fail()
	}

	if contains(env, "MISSING") {
		t.Fail()
	}
}

func TestKeys(t *testing.T) {
	env := map[string]string{
		"VIRTUAL_HOST": "demo.local",
	}
	tests := templateTestList{
		{`{{range (keys $)}}{{.}}{{end}}`, env, `VIRTUAL_HOST`},
	}

	tests.run(t, "keys")
}

func TestKeysEmpty(t *testing.T) {
	input := map[string]int{}

	k, err := keys(input)
	if err != nil {
		t.Fatalf("Error fetching keys: %v", err)
	}
	vk := reflect.ValueOf(k)
	if vk.Kind() == reflect.Invalid {
		t.Fatalf("Got invalid kind for keys: %v", vk)
	}

	if len(input) != vk.Len() {
		t.Fatalf("Incorrect key count; expected %s, got %s", len(input), vk.Len())
	}
}

func TestKeysNil(t *testing.T) {
	k, err := keys(nil)
	if err != nil {
		t.Fatalf("Error fetching keys: %v", err)
	}
	vk := reflect.ValueOf(k)
	if vk.Kind() != reflect.Invalid {
		t.Fatalf("Got invalid kind for keys: %v", vk)
	}
}

func TestIntersect(t *testing.T) {
	if len(intersect([]string{"foo.fo.com", "bar.com"}, []string{"foo.bar.com"})) != 0 {
		t.Fatal("Expected no match")
	}

	if len(intersect([]string{"foo.fo.com", "bar.com"}, []string{"bar.com", "foo.com"})) != 1 {
		t.Fatal("Expected only one match")
	}

	if len(intersect([]string{"foo.com"}, []string{"bar.com", "foo.com"})) != 1 {
		t.Fatal("Expected only one match")
	}

	if len(intersect([]string{"foo.fo.com", "foo.com", "bar.com"}, []string{"bar.com", "foo.com"})) != 2 {
		t.Fatal("Expected two matches")
	}
}

func TestGroupByExistingKey(t *testing.T) {
	containers := []*RuntimeContainer{
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
			},
			ID: "1",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
			},
			ID: "2",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost",
			},
			ID: "3",
		},
	}

	groups, _ := groupBy(containers, "Env.VIRTUAL_HOST")
	if len(groups) != 2 {
		t.Fail()
	}

	if len(groups["demo1.localhost"]) != 2 {
		t.Fail()
	}

	if len(groups["demo2.localhost"]) != 1 {
		t.FailNow()
	}
	if groups["demo2.localhost"][0].(RuntimeContainer).ID != "3" {
		t.Fail()
	}
}

func TestGroupByAfterWhere(t *testing.T) {
	containers := []*RuntimeContainer{
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
				"EXTERNAL":     "true",
			},
			ID: "1",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
			},
			ID: "2",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost",
				"EXTERNAL":     "true",
			},
			ID: "3",
		},
	}

	filtered, _ := where(containers, "Env.EXTERNAL", "true")
	groups, _ := groupBy(filtered, "Env.VIRTUAL_HOST")

	if len(groups) != 2 {
		t.Fail()
	}

	if len(groups["demo1.localhost"]) != 1 {
		t.Fail()
	}

	if len(groups["demo2.localhost"]) != 1 {
		t.FailNow()
	}
	if groups["demo2.localhost"][0].(RuntimeContainer).ID != "3" {
		t.Fail()
	}
}

func TestGroupByMulti(t *testing.T) {
	containers := []*RuntimeContainer{
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
			},
			ID: "1",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost,demo3.localhost",
			},
			ID: "2",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost",
			},
			ID: "3",
		},
	}

	groups, _ := groupByMulti(containers, "Env.VIRTUAL_HOST", ",")
	if len(groups) != 3 {
		t.Fatalf("expected 3 got %d", len(groups))
	}

	if len(groups["demo1.localhost"]) != 2 {
		t.Fatalf("expected 2 got %s", len(groups["demo1.localhost"]))
	}

	if len(groups["demo2.localhost"]) != 1 {
		t.Fatalf("expected 1 got %s", len(groups["demo2.localhost"]))
	}
	if groups["demo2.localhost"][0].(RuntimeContainer).ID != "3" {
		t.Fatalf("expected 2 got %s", groups["demo2.localhost"][0].(RuntimeContainer).ID)
	}
	if len(groups["demo3.localhost"]) != 1 {
		t.Fatalf("expect 1 got %d", len(groups["demo3.localhost"]))
	}
	if groups["demo3.localhost"][0].(RuntimeContainer).ID != "2" {
		t.Fatalf("expected 2 got %s", groups["demo3.localhost"][0].(RuntimeContainer).ID)
	}
}

func TestWhere(t *testing.T) {
	containers := []*RuntimeContainer{
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
			},
			ID: "1",
			Addresses: []Address{
				Address{
					IP:    "172.16.42.1",
					Port:  "80",
					Proto: "tcp",
				},
			},
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost",
			},
			ID: "2",
			Addresses: []Address{
				Address{
					IP:    "172.16.42.1",
					Port:  "9999",
					Proto: "tcp",
				},
			},
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo3.localhost",
			},
			ID: "3",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost",
			},
			ID: "4",
		},
	}

	tests := templateTestList{
		{`{{where . "Env.VIRTUAL_HOST" "demo1.localhost" | len}}`, containers, `1`},
		{`{{where . "Env.VIRTUAL_HOST" "demo2.localhost" | len}}`, containers, `2`},
		{`{{where . "Env.VIRTUAL_HOST" "demo3.localhost" | len}}`, containers, `1`},
		{`{{where . "Env.NOEXIST" "demo3.localhost" | len}}`, containers, `0`},
		{`{{where .Addresses "Port" "80" | len}}`, containers[0], `1`},
		{`{{where .Addresses "Port" "80" | len}}`, containers[1], `0`},
		{
			`{{where . "Value" 5 | len}}`,
			[]struct {
				Value int
			}{
				{Value: 5},
				{Value: 3},
				{Value: 5},
			},
			`2`,
		},
	}

	tests.run(t, "where")
}

func TestWhereExist(t *testing.T) {
	containers := []*RuntimeContainer{
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
				"VIRTUAL_PATH": "/api",
			},
			ID: "1",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost",
			},
			ID: "2",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo3.localhost",
				"VIRTUAL_PATH": "/api",
			},
			ID: "3",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_PROTO": "https",
			},
			ID: "4",
		},
	}

	tests := templateTestList{
		{`{{whereExist . "Env.VIRTUAL_HOST" | len}}`, containers, `3`},
		{`{{whereExist . "Env.VIRTUAL_PATH" | len}}`, containers, `2`},
		{`{{whereExist . "Env.NOT_A_KEY" | len}}`, containers, `0`},
		{`{{whereExist . "Env.VIRTUAL_PROTO" | len}}`, containers, `1`},
	}

	tests.run(t, "whereExist")
}

func TestWhereNotExist(t *testing.T) {
	containers := []*RuntimeContainer{
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
				"VIRTUAL_PATH": "/api",
			},
			ID: "1",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost",
			},
			ID: "2",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo3.localhost",
				"VIRTUAL_PATH": "/api",
			},
			ID: "3",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_PROTO": "https",
			},
			ID: "4",
		},
	}

	tests := templateTestList{
		{`{{whereNotExist . "Env.VIRTUAL_HOST" | len}}`, containers, `1`},
		{`{{whereNotExist . "Env.VIRTUAL_PATH" | len}}`, containers, `2`},
		{`{{whereNotExist . "Env.NOT_A_KEY" | len}}`, containers, `4`},
		{`{{whereNotExist . "Env.VIRTUAL_PROTO" | len}}`, containers, `3`},
	}

	tests.run(t, "whereNotExist")
}

func TestWhereSomeMatch(t *testing.T) {
	containers := []*RuntimeContainer{
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
			},
			ID: "1",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost,demo4.localhost",
			},
			ID: "2",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "bar,demo3.localhost,foo",
			},
			ID: "3",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost",
			},
			ID: "4",
		},
	}

	tests := templateTestList{
		{`{{whereAny . "Env.VIRTUAL_HOST" "," (split "demo1.localhost" ",") | len}}`, containers, `1`},
		{`{{whereAny . "Env.VIRTUAL_HOST" "," (split "demo2.localhost,lala" ",") | len}}`, containers, `2`},
		{`{{whereAny . "Env.VIRTUAL_HOST" "," (split "something,demo3.localhost" ",") | len}}`, containers, `1`},
		{`{{whereAny . "Env.NOEXIST" "," (split "demo3.localhost" ",") | len}}`, containers, `0`},
	}

	tests.run(t, "whereAny")
}

func TestWhereRequires(t *testing.T) {
	containers := []*RuntimeContainer{
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
			},
			ID: "1",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost,demo4.localhost",
			},
			ID: "2",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "bar,demo3.localhost,foo",
			},
			ID: "3",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost",
			},
			ID: "4",
		},
	}

	tests := templateTestList{
		{`{{whereAll . "Env.VIRTUAL_HOST" "," (split "demo1.localhost" ",") | len}}`, containers, `1`},
		{`{{whereAll . "Env.VIRTUAL_HOST" "," (split "demo2.localhost,lala" ",") | len}}`, containers, `0`},
		{`{{whereAll . "Env.VIRTUAL_HOST" "," (split "demo3.localhost" ",") | len}}`, containers, `1`},
		{`{{whereAll . "Env.NOEXIST" "," (split "demo3.localhost" ",") | len}}`, containers, `0`},
	}

	tests.run(t, "whereAll")
}

func TestHasPrefix(t *testing.T) {
	const prefix = "tcp://"
	const str = "tcp://127.0.0.1:2375"
	if !hasPrefix(prefix, str) {
		t.Fatalf("expected %s to have prefix %s", str, prefix)
	}
}

func TestHasSuffix(t *testing.T) {
	const suffix = ".local"
	const str = "myhost.local"
	if !hasSuffix(suffix, str) {
		t.Fatalf("expected %s to have suffix %s", str, suffix)
	}
}

func TestTrimPrefix(t *testing.T) {
	const prefix = "tcp://"
	const str = "tcp://127.0.0.1:2375"
	const trimmed = "127.0.0.1:2375"
	got := trimPrefix(prefix, str)
	if got != trimmed {
		t.Fatalf("expected trimPrefix(%s,%s) to be %s, got %s", prefix, str, trimmed, got)
	}
}

func TestTrimSuffix(t *testing.T) {
	const suffix = ".local"
	const str = "myhost.local"
	const trimmed = "myhost"
	got := trimSuffix(suffix, str)
	if got != trimmed {
		t.Fatalf("expected trimSuffix(%s,%s) to be %s, got %s", suffix, str, trimmed, got)
	}
}

func TestTrim(t *testing.T) {
	const str = "  myhost.local  "
	const trimmed = "myhost.local"
	got := trim(str)
	if got != trimmed {
		t.Fatalf("expected trim(%s) to be %s, got %s", str, trimmed, got)
	}
}

func TestDict(t *testing.T) {
	containers := []*RuntimeContainer{
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
			},
			ID: "1",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost,demo3.localhost",
			},
			ID: "2",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost",
			},
			ID: "3",
		},
	}
	d, err := dict("/", containers)
	if err != nil {
		t.Fatal(err)
	}
	if d["/"] == nil {
		t.Fatalf("did not find containers in dict: %s", d)
	}
	if d["MISSING"] != nil {
		t.Fail()
	}
}

func TestSha1(t *testing.T) {
	sum := hashSha1("/path")
	if sum != "4f26609ad3f5185faaa9edf1e93aa131e2131352" {
		t.Fatal("Incorrect SHA1 sum")
	}
}

func TestJson(t *testing.T) {
	containers := []*RuntimeContainer{
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
			},
			ID: "1",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost,demo3.localhost",
			},
			ID: "2",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost",
			},
			ID: "3",
		},
	}
	output, err := marshalJson(containers)
	if err != nil {
		t.Fatal(err)
	}

	buf := bytes.NewBufferString(output)
	dec := json.NewDecoder(buf)
	if err != nil {
		t.Fatal(err)
	}
	var decoded []*RuntimeContainer
	if err := dec.Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded) != len(containers) {
		t.Fatal("Incorrect unmarshaled container count. Expected %d, got %d.", len(containers), len(decoded))
	}
}

func TestParseJson(t *testing.T) {
	tests := templateTestList{
		{`{{parseJson .}}`, `null`, `<no value>`},
		{`{{parseJson .}}`, `true`, `true`},
		{`{{parseJson .}}`, `1`, `1`},
		{`{{parseJson .}}`, `0.5`, `0.5`},
		{`{{index (parseJson .) "enabled"}}`, `{"enabled":true}`, `true`},
		{`{{index (parseJson . | first) "enabled"}}`, `[{"enabled":true}]`, `true`},
	}

	tests.run(t, "parseJson")
}

func TestQueryEscape(t *testing.T) {
	tests := templateTestList{
		{`{{queryEscape .}}`, `example.com`, `example.com`},
		{`{{queryEscape .}}`, `.example.com`, `.example.com`},
		{`{{queryEscape .}}`, `*.example.com`, `%2A.example.com`},
		{`{{queryEscape .}}`, `~^example\.com(\..*\.xip\.io)?$`, `~%5Eexample%5C.com%28%5C..%2A%5C.xip%5C.io%29%3F%24`},
	}

	tests.run(t, "queryEscape")
}

func TestArrayClosestExact(t *testing.T) {
	if arrayClosest([]string{"foo.bar.com", "bar.com"}, "foo.bar.com") != "foo.bar.com" {
		t.Fatal("Expected foo.bar.com")
	}
}

func TestArrayClosestSubstring(t *testing.T) {
	if arrayClosest([]string{"foo.fo.com", "bar.com"}, "foo.bar.com") != "bar.com" {
		t.Fatal("Expected bar.com")
	}
}

func TestArrayClosestNoMatch(t *testing.T) {
	if arrayClosest([]string{"foo.fo.com", "bip.com"}, "foo.bar.com") != "" {
		t.Fatal("Expected ''")
	}
}
