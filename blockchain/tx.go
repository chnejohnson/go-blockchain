package blockchain

// TxInput is just a reference to previous output
type TxInput struct {
	ID  []byte // the transaction that output is inside of
	Out int    // the index where this output appear
	Sig string // a script which provide a data used in the output pubkey, such as user account
}

// TxOutput has Value which is the transaction token,
// and the Pubkey is for unlocking the Value field.
// In bitcoin, the Pubkey is derived from a complicated scripting language called 'script'
// we just use arbitrary key to represent the user's address
type TxOutput struct {
	Value  int    // how much token be send
	Pubkey string // the token receiver's address
}

// CanUnlock check if data is input's signature
func (in *TxInput) CanUnlock(data string) bool {
	return in.Sig == data
}

// CanBeUnlocked check if data is output's Pubkey
func (out *TxOutput) CanBeUnlocked(data string) bool {
	return out.Pubkey == data
}
