package endpoint

func NewRpcRequestContext() *RpcRequestContext {
	return &RpcRequestContext{
		stringParams: make(map[string]string),
		intParams:    make(map[string]int64),
		boolParams:   make(map[string]bool),
	}
}

type RpcRequestContext struct {
	stringParams map[string]string
	intParams    map[string]int64
	boolParams   map[string]bool
	apiToken     string
	authorized   bool
}

func (r *RpcRequestContext) GetString(key string) (value string, err error) {
	if v, found := r.stringParams[key]; found {
		return v, nil
	}
	return "", errKeyNotFound
}

func (r *RpcRequestContext) GetInt(key string) (value int64, err error) {
	if v, found := r.intParams[key]; found {
		return v, nil
	}
	return 0, errKeyNotFound
}

func (r *RpcRequestContext) GetBool(key string) (value bool, err error) {
	if v, found := r.boolParams[key]; found {
		return v, nil
	}
	return false, errKeyNotFound
}

func (r *RpcRequestContext) GetApiToken() (token string, err error) {
	return r.apiToken, nil
}

func (r *RpcRequestContext) SetBool(key string, value bool) {
	r.boolParams[key] = value
}

func (r *RpcRequestContext) SetString(key, value string) {
	r.stringParams[key] = value
}

func (r *RpcRequestContext) SetInt(key string, value int64) {
	r.intParams[key] = value
}

func (r *RpcRequestContext) Authorized(v bool) {
	r.authorized = v
}
