package orm

import "context"

// BeforeCreateWithContextHook is invoked before a model is inserted, with context.
type BeforeCreateWithContextHook interface {
	BeforeCreate(ctx context.Context) error
}

// AfterCreateWithContextHook is invoked after a model is inserted, with context.
type AfterCreateWithContextHook interface {
	AfterCreate(ctx context.Context) error
}

// BeforeUpdateWithContextHook is invoked before a model is updated, with context.
type BeforeUpdateWithContextHook interface {
	BeforeUpdate(ctx context.Context) error
}

// AfterUpdateWithContextHook is invoked after a model is updated, with context.
type AfterUpdateWithContextHook interface {
	AfterUpdate(ctx context.Context) error
}

// BeforeDeleteWithContextHook is invoked before a model is deleted, with context.
type BeforeDeleteWithContextHook interface {
	BeforeDelete(ctx context.Context) error
}

// AfterDeleteWithContextHook is invoked after a model is deleted, with context.
type AfterDeleteWithContextHook interface {
	AfterDelete(ctx context.Context) error
}

// AfterFindWithContextHook is invoked after a model is fetched from database, with context.
type AfterFindWithContextHook interface {
	AfterFind(ctx context.Context) error
}

// BeforeCreateHook is invoked before a model is inserted.
type BeforeCreateHook interface {
	BeforeCreate() error
}

// AfterCreateHook is invoked after a model is inserted.
type AfterCreateHook interface {
	AfterCreate() error
}

// BeforeUpdateHook is invoked before a model is updated.
type BeforeUpdateHook interface {
	BeforeUpdate() error
}

// AfterUpdateHook is invoked after a model is updated.
type AfterUpdateHook interface {
	AfterUpdate() error
}

// BeforeDeleteHook is invoked before a model is deleted.
type BeforeDeleteHook interface {
	BeforeDelete() error
}

// AfterDeleteHook is invoked after a model is deleted.
type AfterDeleteHook interface {
	AfterDelete() error
}

// AfterFindHook is invoked after a model is fetched from database.
type AfterFindHook interface {
	AfterFind() error
}

func invokeBeforeCreate(ctx context.Context, m any) error {
	if h, ok := m.(BeforeCreateWithContextHook); ok {
		return h.BeforeCreate(ctx)
	}
	if h, ok := m.(BeforeCreateHook); ok {
		return h.BeforeCreate()
	}
	return nil
}

func invokeAfterCreate(ctx context.Context, m any) error {
	if h, ok := m.(AfterCreateWithContextHook); ok {
		return h.AfterCreate(ctx)
	}
	if h, ok := m.(AfterCreateHook); ok {
		return h.AfterCreate()
	}
	return nil
}

func invokeBeforeUpdate(ctx context.Context, m any) error {
	if h, ok := m.(BeforeUpdateWithContextHook); ok {
		return h.BeforeUpdate(ctx)
	}
	if h, ok := m.(BeforeUpdateHook); ok {
		return h.BeforeUpdate()
	}
	return nil
}

func invokeAfterUpdate(ctx context.Context, m any) error {
	if h, ok := m.(AfterUpdateWithContextHook); ok {
		return h.AfterUpdate(ctx)
	}
	if h, ok := m.(AfterUpdateHook); ok {
		return h.AfterUpdate()
	}
	return nil
}

func invokeBeforeDelete(ctx context.Context, m any) error {
	if h, ok := m.(BeforeDeleteWithContextHook); ok {
		return h.BeforeDelete(ctx)
	}
	if h, ok := m.(BeforeDeleteHook); ok {
		return h.BeforeDelete()
	}
	return nil
}

func invokeAfterDelete(ctx context.Context, m any) error {
	if h, ok := m.(AfterDeleteWithContextHook); ok {
		return h.AfterDelete(ctx)
	}
	if h, ok := m.(AfterDeleteHook); ok {
		return h.AfterDelete()
	}
	return nil
}

func invokeAfterFind(ctx context.Context, m any) error {
	if h, ok := m.(AfterFindWithContextHook); ok {
		return h.AfterFind(ctx)
	}
	if h, ok := m.(AfterFindHook); ok {
		return h.AfterFind()
	}
	return nil
}
