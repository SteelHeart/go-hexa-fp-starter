package domain

import "github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"

// RegistrationCommand is a registration intent as it arrives from a surface:
// primitive types, not yet validated.
//
// This is the ONLY place in the domain where bare `string`s are found, and that
// is exactly this type's role: to mark the boundary.
type RegistrationCommand struct {
	Email    string
	Password string
}

// ValidRegistration is a validated registration intent.
//
// Once this type is obtained, no further validation is needed downstream: the
// boundary crossing happened, once, here.
type ValidRegistration struct {
	Email    Email
	Password RawPassword
}

// ParseRegistration validates a registration command.
//
// Pure function: no I/O, no clock. Address uniqueness is NOT checked here — that
// is an effect, therefore a port, called by the use case.
func ParseRegistration(cmd RegistrationCommand) result.Result[ValidRegistration, Error] {
	// Order matters: the email address is reported before the password,
	// because it is the field the user fills in first.
	return result.FlatMap(
		NewEmail(cmd.Email),
		func(email Email) result.Result[ValidRegistration, Error] {
			return result.Map(
				NewRawPassword(cmd.Password),
				func(password RawPassword) ValidRegistration {
					return ValidRegistration{Email: email, Password: password}
				},
			)
		},
	)
}
