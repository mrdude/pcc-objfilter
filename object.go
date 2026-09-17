package objfilter

type Object interface {
	// GetValue returns a Value for the
	// given property name
	GetValue(name string) Value
}

type cachingObject struct {
	delegate Object
	cache    map[string]Value
}

func newCachingObject(obj Object) Object {
	if _, ok := obj.(*cachingObject); ok {
		return obj
	}

	return &cachingObject{
		delegate: obj,
		cache:    make(map[string]Value),
	}
}

func (c *cachingObject) GetValue(name string) Value {
	// check the cache
	val, ok := c.cache[name]
	if ok {
		return val
	}

	// check the object
	val = c.delegate.GetValue(name)

	// fill the cache
	c.cache[name] = val

	// return
	return val
}
