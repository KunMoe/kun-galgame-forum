package apiv1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

// huma gives only struct types a component and inlines every enum, so a code
// generator saw each use of a vocabulary as a new type: the App's tonik spike
// made 1284 Dart types, 829 of them named by position (WorksParametersModel3),
// and ResourceLanguage alone came out as 12 incompatible enums. Only the
// published document is rewritten; request validation and the v1 gates still
// read huma's own.
func publishSpec(raw []byte) ([]byte, error) {
	return publishWith(raw, vocabularies)
}

func publishWith(raw []byte, vocabs []vocabulary) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	root, err := decodeOrdered(dec)
	if err != nil {
		return nil, err
	}
	doc, ok := root.(*orderedObject)
	if !ok {
		return nil, fmt.Errorf("publish: the document is not an object")
	}
	p, err := newPublisher(doc, vocabs)
	if err != nil {
		return nil, err
	}
	p.walkDocument()
	if err := p.missingError(); err != nil {
		return nil, err
	}
	p.addComponents()
	var buf bytes.Buffer
	if err := encodeOrdered(&buf, doc); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type vocabulary struct {
	Name   string
	Values []string
}

func vocabularyKey(values []string) string {
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	return strings.Join(sorted, "\x00")
}

type publisher struct {
	doc     *orderedObject
	schemas *orderedObject
	byKey   map[string]vocabulary
	used    map[string]vocabulary
	missing map[string][]string
}

func newPublisher(doc *orderedObject, vocabs []vocabulary) (*publisher, error) {
	schemas := doc.object("components").object("schemas")
	if schemas == nil {
		return nil, fmt.Errorf("publish: components.schemas is missing")
	}
	p := &publisher{
		doc:     doc,
		schemas: schemas,
		byKey:   map[string]vocabulary{},
		used:    map[string]vocabulary{},
		missing: map[string][]string{},
	}
	for _, v := range vocabs {
		if _, taken := schemas.vals[v.Name]; taken {
			return nil, fmt.Errorf("publish: vocabulary %s collides with a component of the same name", v.Name)
		}
		key := vocabularyKey(v.Values)
		if prev, dup := p.byKey[key]; dup {
			return nil, fmt.Errorf("publish: vocabularies %s and %s have the same values", prev.Name, v.Name)
		}
		p.byKey[key] = v
	}
	return p, nil
}

func (p *publisher) walkDocument() {
	for _, name := range p.schemas.keys {
		p.schemas.vals[name] = p.schema(p.schemas.vals[name], name)
	}
	paths := p.doc.object("paths")
	if paths == nil {
		return
	}
	for _, path := range paths.keys {
		item := paths.object(path)
		if item == nil {
			continue
		}
		for _, method := range item.keys {
			op := item.object(method)
			if op == nil {
				continue
			}
			where := method + " " + path
			if params, ok := op.vals["parameters"].([]any); ok {
				for _, raw := range params {
					if param, ok := raw.(*orderedObject); ok {
						p.replace(param, "schema", fmt.Sprintf("%s ?%v", where, param.vals["name"]))
					}
				}
			}
			p.content(op.object("requestBody"), where+" body")
			if responses := op.object("responses"); responses != nil {
				for _, status := range responses.keys {
					resp := responses.object(status)
					p.content(resp, where+" "+status)
					if headers := resp.object("headers"); headers != nil {
						for _, h := range headers.keys {
							p.replace(headers.object(h), "schema", where+" "+status+" "+h)
						}
					}
				}
			}
		}
	}
}

func (p *publisher) content(o *orderedObject, where string) {
	content := o.object("content")
	if content == nil {
		return
	}
	for _, ct := range content.keys {
		p.replace(content.object(ct), "schema", where)
	}
}

func (p *publisher) replace(o *orderedObject, key, where string) {
	if o == nil {
		return
	}
	if v, ok := o.vals[key]; ok {
		o.vals[key] = p.schema(v, where)
	}
}

func (p *publisher) schema(v any, where string) any {
	o, ok := v.(*orderedObject)
	if !ok {
		return v
	}
	if props := o.object("properties"); props != nil {
		for _, name := range props.keys {
			props.vals[name] = p.schema(props.vals[name], where+"."+name)
		}
	}
	for _, key := range []string{"items", "additionalProperties", "not"} {
		p.replace(o, key, where)
	}
	for _, key := range []string{"allOf", "anyOf", "oneOf", "prefixItems"} {
		if list, ok := o.vals[key].([]any); ok {
			for i := range list {
				list[i] = p.schema(list[i], where)
			}
		}
	}
	if enum, ok := o.vals["enum"].([]any); ok {
		return p.enum(o, enum, where)
	}
	return o
}

func (p *publisher) enum(o *orderedObject, enum []any, where string) any {
	var values []string
	nullable := false
	for _, e := range enum {
		switch x := e.(type) {
		case nil:
			nullable = true
		case string:
			values = append(values, x)
		default:
			return o
		}
	}
	if len(values) == 1 && !nullable {
		o.remove("enum")
		o.remove("x-vocabulary-closed")
		o.set("const", values[0])
		o.sortKeys()
		return o
	}
	v, ok := p.byKey[vocabularyKey(values)]
	if !ok {
		key := vocabularyKey(values)
		p.missing[key] = append(p.missing[key], where)
		return o
	}
	p.used[v.Name] = v
	out := newOrderedObject()
	ref := newOrderedObject()
	ref.set("$ref", schemaRefPrefix+v.Name)
	if nullable {
		null := newOrderedObject()
		null.set("type", "null")
		out.set("anyOf", []any{ref, null})
	} else {
		out.set("$ref", schemaRefPrefix+v.Name)
	}
	for _, key := range []string{"default", "description"} {
		if val, ok := o.vals[key]; ok {
			out.set(key, val)
		}
	}
	out.sortKeys()
	return out
}

func (p *publisher) missingError() error {
	if len(p.missing) == 0 {
		return nil
	}
	keys := make([]string, 0, len(p.missing))
	for k := range p.missing {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	var b strings.Builder
	b.WriteString("publish: closed vocabularies without a name; add each to publish_vocabularies.go:")
	for _, k := range keys {
		fmt.Fprintf(&b, "\n  {%s} at %s", strings.ReplaceAll(k, "\x00", ","), strings.Join(p.missing[k], "; "))
	}
	return fmt.Errorf("%s", b.String())
}

func (p *publisher) addComponents() {
	for name, v := range p.used {
		enum := make([]any, len(v.Values))
		longest := 0
		for i, s := range v.Values {
			enum[i] = s
			longest = max(longest, utf8.RuneCountInString(s))
		}
		c := newOrderedObject()
		c.set("enum", enum)
		c.set("maxLength", json.Number(fmt.Sprint(longest)))
		c.set("type", "string")
		c.set("x-vocabulary-closed", true)
		p.schemas.set(name, c)
	}
	p.schemas.sortKeys()
}

// orderedObject keeps member order, so the published spec differs from what
// huma wrote only where publishing changed it.
type orderedObject struct {
	keys []string
	vals map[string]any
}

func newOrderedObject() *orderedObject {
	return &orderedObject{vals: map[string]any{}}
}

func (o *orderedObject) object(key string) *orderedObject {
	if o == nil {
		return nil
	}
	v, _ := o.vals[key].(*orderedObject)
	return v
}

func (o *orderedObject) set(key string, v any) {
	if _, ok := o.vals[key]; !ok {
		o.keys = append(o.keys, key)
	}
	o.vals[key] = v
}

func (o *orderedObject) remove(key string) {
	if _, ok := o.vals[key]; !ok {
		return
	}
	delete(o.vals, key)
	o.keys = slices.DeleteFunc(o.keys, func(k string) bool { return k == key })
}

func (o *orderedObject) sortKeys() {
	slices.Sort(o.keys)
}

func decodeOrdered(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		return tok, nil
	}
	switch delim {
	case '{':
		o := newOrderedObject()
		for dec.More() {
			kt, err := dec.Token()
			if err != nil {
				return nil, err
			}
			key, _ := kt.(string)
			v, err := decodeOrdered(dec)
			if err != nil {
				return nil, err
			}
			o.set(key, v)
		}
		_, err := dec.Token()
		return o, err
	case '[':
		list := []any{}
		for dec.More() {
			v, err := decodeOrdered(dec)
			if err != nil {
				return nil, err
			}
			list = append(list, v)
		}
		_, err := dec.Token()
		return list, err
	}
	return nil, fmt.Errorf("publish: unexpected %v", delim)
}

func encodeOrdered(buf *bytes.Buffer, v any) error {
	switch x := v.(type) {
	case *orderedObject:
		buf.WriteByte('{')
		for i, k := range x.keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := encodeOrdered(buf, k); err != nil {
				return err
			}
			buf.WriteByte(':')
			if err := encodeOrdered(buf, x.vals[k]); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
	case []any:
		buf.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := encodeOrdered(buf, e); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	default:
		b, err := json.Marshal(x)
		if err != nil {
			return err
		}
		buf.Write(b)
	}
	return nil
}

func PublishedVocabularies() []string {
	names := make([]string, len(vocabularies))
	for i, v := range vocabularies {
		names[i] = v.Name
	}
	return names
}
