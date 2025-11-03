package swag

const Name = "swagger"

type Doc interface {
	ReadDoc() string
}

var docs = make(map[string]Doc)

func Register(name string, doc Doc) {
	docs[name] = doc
}

func ReadDoc(name string) string {
	if doc, ok := docs[name]; ok {
		return doc.ReadDoc()
	}
	return ""
}
