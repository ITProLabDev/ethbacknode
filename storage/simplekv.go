package storage

type Key interface {
	GetKey() []byte
}

type Data interface {
	Key
	Encode() []byte
	Decode([]byte) error
}

type SimpleStorage interface {
	Save(data Data) (err error)
	Read(key Key, data Data) (err error)
	ReadAll(processor func(raw []byte) (err error)) (err error)
	Delete(rowKey []byte) (err error)
}

type SimpleKeyStorage interface {
	Save(data Data) (err error)
	Read(key Key, data Data) (err error)
	ReadAll(processor func(raw []byte) (err error)) (err error)
	ReadAllKey(processor func(key, raw []byte) (err error)) (err error)
	Delete(rowKey []byte) (err error)
}
