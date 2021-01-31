package reconcile

type InformerOptions struct {
	List bool
}

type InformerOption func(o *InformerOptions)

func WithList(list bool) InformerOption {
	return func(o *InformerOptions) {
		o.List = list
	}
}
