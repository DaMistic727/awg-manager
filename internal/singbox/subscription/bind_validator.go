package subscription

import (
	"context"
	"errors"
	"fmt"
)

// ErrBindNotSelectable — имени нет в каталоге выбираемых интерфейсов: ошибка
// ввода, в отличие от сбоя чтения каталога. Валидатор оборачивает ею отказ,
// сервис отвечает на неё ErrInvalidInput (400), а не 500 (F429).
var ErrBindNotSelectable = errors.New("bind_interface is not a selectable interface")

// BindInterfaceValidator checks egress interface names against the router's
// bindable-interface catalog. Optional on Service — when nil, bind is not
// validated (tests / legacy bootstrap).
type BindInterfaceValidator interface {
	ValidateBindInterface(ctx context.Context, name string) error
}

func validateBindInterfaceOptional(ctx context.Context, v BindInterfaceValidator, name string) error {
	if name == "" || v == nil {
		return nil
	}
	err := v.ValidateBindInterface(ctx, name)
	if errors.Is(err, ErrBindNotSelectable) {
		return fmt.Errorf("%w: %w", ErrInvalidInput, err)
	}
	return err
}
