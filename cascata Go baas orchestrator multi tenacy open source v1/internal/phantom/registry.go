package phantom

// Source defines where an extension payload can be extracted from.
type Source struct {
	Name        string
	Image       string
	Provides    []string
	EstimateMB  int
	Description string
}

// ExtensionRegistry maps extension slug to its Phantom Source.
var ExtensionRegistry = map[string]Source{
	"vector": {
		Name:        "pgvector",
		Image:       "pgvector/pgvector:0.8.0-pg18",
		Provides:    []string{"vector"},
		EstimateMB:  5,
		Description: "pgvector — AI/RAG vector embeddings",
	},
	"postgis": {
		Name:        "postgis",
		Image:       "postgis/postgis:18-3.6-alpine",
		Provides:    []string{"postgis", "postgis_tiger_geocoder", "postgis_topology", "address_standardizer", "address_standardizer_data_us"},
		EstimateMB:  80,
		Description: "PostGIS — Geospatial functions",
	},
	"timescaledb": {
		Name:        "timescaledb",
		Image:       "timescale/timescaledb-ha:pg18",
		Provides:    []string{"timescaledb"},
		EstimateMB:  35,
		Description: "TimescaleDB — Time-series data",
	},
}

// IsPhantom checks if an extension requires out-of-band injection.
func IsPhantom(name string) (Source, bool) {
	s, ok := ExtensionRegistry[name]
	return s, ok
}
