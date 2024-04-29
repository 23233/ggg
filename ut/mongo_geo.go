package ut

// mongodb的经纬度格式需要 需要建立索引 2dsphere 默认是wgs84

type Geo struct {
	Type        string    `json:"type" bson:"type,omitempty" mapstructure:"type"` // Point
	Coordinates []float64 `json:"coordinates" bson:"coordinates,omitempty"`       // lng,lat
}

func NewGeo(lat, long float64) Geo {
	return Geo{
		"Point",
		[]float64{long, lat},
	}
}
