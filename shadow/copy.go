package shadow

// cloneState creates deepcopy of thing state.
func cloneState(s map[string]interface{}) map[string]interface{} {
	c := map[string]interface{}{}
	for k, v := range s {
		switch vv := v.(type) {
		case map[string]interface{}:
			c[k] = cloneState(vv)
		case []interface{}:
			c[k] = cloneStateSlice(vv)
		default:
			c[k] = vv
		}
	}
	return c
}

// cloneStateSlice creates deepcopy of thing state slice.
func cloneStateSlice(s []interface{}) []interface{} {
	c := []interface{}{}
	for _, v := range s {
		switch vv := v.(type) {
		case map[string]interface{}:
			c = append(c, cloneState(vv))
		case []interface{}:
			c = append(c, cloneStateSlice(vv))
		default:
			c = append(c, vv)
		}
	}
	return c
}

// clone returns deepcopy of the document.
func (s *ThingDocument) clone() *ThingDocument {
	c := *s
	c.State.Desired = cloneState(s.State.Desired)
	c.State.Reported = cloneState(s.State.Reported)
	c.State.Delta = cloneState(s.State.Delta)
	return &c
}
