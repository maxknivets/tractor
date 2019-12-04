package manifold

type attributeset map[string]interface{}

func (s attributeset) HasAttribute(attr string) bool {
	_, ok := s[attr]
	return ok
}

func (s attributeset) GetAttribute(attr string) interface{} {
	return s[attr]
}

func (s attributeset) SetAttribute(attr string, value interface{}) {
	s[attr] = value
}

func (s attributeset) UnsetAttribute(attr string) {
	delete(s, attr)
}
