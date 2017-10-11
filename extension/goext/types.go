package goext

type MaybeState int

const (
	MaybeUndefined MaybeState = iota
	MaybeNull
	MaybeNotNull
)

// MaybeString represents 3-valued string
type MaybeString struct {
	State MaybeState
	Value string
}

// MaybeFloat represents 3-valued float
type MaybeFloat struct {
	State MaybeState
	Value float64
}

// MaybeInt represents 3-valued int
type MaybeInt struct {
	State MaybeState
	Value int
}

// MaybeBool represents 3-valued bool
type MaybeBool struct {
	State MaybeState
	Value bool
}

// IsUndefined returns whether value is undefined
func (ms MaybeString) IsUndefined() bool {
	return ms.State == MaybeUndefined
}

// IsNull returns whether value is null
func (ms MaybeString) IsNull() bool {
	return ms.State == MaybeNull
}

// IsNotNull returns whether value is defined and not null
func (ms MaybeString) IsNotNull() bool {
	return ms.State == MaybeNotNull
}

// IsUndefined returns whether value is undefined
func (mb MaybeBool) IsUndefined() bool {
	return mb.State == MaybeUndefined
}

// IsNull returns whether value is null
func (mb MaybeBool) IsNull() bool {
	return mb.State == MaybeNull
}

// IsNotNull returns whether value is defined and not null
func (mb MaybeBool) IsNotNull() bool {
	return mb.State == MaybeNotNull
}

// IsUndefined returns whether value is undefined
func (mi MaybeInt) IsUndefined() bool {
	return mi.State == MaybeUndefined
}

// IsNull returns whether value is null
func (mi MaybeInt) IsNull() bool {
	return mi.State == MaybeNull
}

// IsNotNull returns whether value is defined and not null
func (mi MaybeInt) IsNotNull() bool {
	return mi.State == MaybeNotNull
}

// IsUndefined returns whether value is undefined
func (mf MaybeFloat) IsUndefined() bool {
	return mf.State == MaybeUndefined
}

// IsNull returns whether value is null
func (mf MaybeFloat) IsNull() bool {
	return mf.State == MaybeNull
}

// IsNotNull returns whether value is defined and not null
func (mf MaybeFloat) IsNotNull() bool {
	return mf.State == MaybeNotNull
}

/*
   Equality rules:
   https://developer.mozilla.org/en-US/docs/Web/JavaScript/Equality_comparisons_and_sameness

   |-----------|-----------|-------|-------|
   | Operands  | Undefined | Null  | Value |
   |-----------|-----------|-------|-------|
   | Undefined | true      | true  | false |
   | Null      | true      | true  | false |
   | Value     | false     | false | A==B  |
   |-----------|-----------|-------|-------|

*/

// Equals returns whether two maybe values are equal
func (this MaybeFloat) Equals(other MaybeFloat) bool {
	if this.IsNotNull() && other.IsNotNull() {
		return this.Value == other.Value
	}
	return !this.IsNotNull() && !other.IsNotNull()
}

// Equals returns whether two maybe values are equal
func (this MaybeString) Equals(other MaybeString) bool {
	if this.IsNotNull() && other.IsNotNull() {
		return this.Value == other.Value
	}
	return !this.IsNotNull() && !other.IsNotNull()
}

// Equals returns whether two maybe values are equal
func (this MaybeBool) Equals(other MaybeBool) bool {
	if this.IsNotNull() && other.IsNotNull() {
		return this.Value == other.Value
	}
	return !this.IsNotNull() && !other.IsNotNull()
}

// Equals returns whether two maybe values are equal
func (this MaybeInt) Equals(other MaybeInt) bool {
	if this.IsNotNull() && other.IsNotNull() {
		return this.Value == other.Value
	}
	return !this.IsNotNull() && !other.IsNotNull()
}

// MakeNullString allocates a new null string
func MakeNullString() MaybeString {
	return MaybeString{
		State: MaybeNull,
	}
}

// MakeNullInt allocates a new null integer
func MakeNullInt() MaybeInt {
	return MaybeInt{
		State: MaybeNull,
	}
}

// MakeNullBool allocates a new null bool
func MakeNullBool() MaybeBool {
	return MaybeBool{
		State: MaybeNull,
	}
}

// MakeNullFloat allocates a new null float
func MakeNullFloat() MaybeFloat {
	return MaybeFloat{
		State: MaybeNull,
	}
}

// MakeMaybeString allocates a new MaybeString and sets its value
func MakeMaybeString(value string) MaybeString {
	return MaybeString{
		Value: value,
		State: MaybeNotNull,
	}
}

// MakeMaybeInt allocates a new MaybeInt and sets its value
func MakeMaybeInt(value int) MaybeInt {
	return MaybeInt{
		Value: value,
		State: MaybeNotNull,
	}
}

// MakeMaybeFloat allocates a new MaybeFloat and sets its value
func MakeMaybeFloat(value float64) MaybeFloat {
	return MaybeFloat{
		Value: value,
		State: MaybeNotNull,
	}
}

// MakeMaybeBool allocates a new MaybeBool and sets its value
func MakeMaybeBool(value bool) MaybeBool {
	return MaybeBool{
		Value: value,
		State: MaybeNotNull,
	}
}
