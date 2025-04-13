package blobcache

type Coordinator struct {
}

func NewCoordinator() *Coordinator {
	return &Coordinator{}
}

func (c *Coordinator) Start() error {
	return nil
}
