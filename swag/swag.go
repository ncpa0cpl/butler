package swag

import (
	"fmt"

	"github.com/zeebo/xxh3"

	"github.com/gofrs/uuid"
)

type CacheOptions struct {
	DisableAutoResponseSkipping bool
	DisableETagGeneration       bool
	MaxAge                      string
	SMaxAge                     string
	StaleWhileRevalidate        string
	StaleIfError                string
	Immutable                   bool
	NoStore                     bool
	NoCache                     bool
	MustRevalidate              bool
	Private                     bool
	ProxyRevalidate             bool
	MustUnderstand              bool
	NoTransform                 bool
	ExampleHeader               string
}

type EndpointData struct {
	Uid             string
	Name            string
	Description     string
	Children        []EndpointData
	Method          string
	Path            string
	IsWs            bool
	IsGroup         bool
	RespContentType string
	CacheOptions    *CacheOptions
	Encoding        string
	ParamsT         TypeStructure
	BodyT           TypeStructure
	ResponseT       TypeStructure
}

func CreateApiDocumentation(endpoints []EndpointData) ([]byte, error) {
	html, err := generateDocPage(endpoints)

	if err != nil {
		return nil, fmt.Errorf("failed to generate a api doc page: %v", err)
	}

	return html, nil
}

func (e *EndpointData) PopulateUid(level int) {
	deterministicStr := fmt.Sprintf("[%v];methd:%s;path:%s;name:%s;description:%s;ws:%v;group:%v;ctype:%s;encoding:%s", level, e.Method, e.Path, e.Name, e.Description, e.IsWs, e.IsGroup, e.RespContentType, e.Encoding)

	var hash [16]byte
	var guid uuid.UUID

	hash = xxh3.HashString128(deterministicStr).Bytes()

	// uuid.FromBytes returns an error if the slice
	// of bytes is not 16 - as hash is defined as
	// [16]byte then we can ignore checking the error
	guid, _ = uuid.FromBytes(hash[:])
	e.Uid = guid.String()
}
