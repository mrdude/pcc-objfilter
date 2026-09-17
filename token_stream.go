package objfilter

type tokenStream struct {
	tokCh     <-chan token
	lookahead int
	buf       []token
	eofInBuf  bool // if true, EOF has been fill()-ed to the buffer
}

func newTokenStream(tokCh <-chan token, lookahead int) *tokenStream {
	return &tokenStream{tokCh: tokCh, lookahead: lookahead}
}

// fill the buffer from the token stream
func (ts *tokenStream) fill(count int) {
	for i := 0; i < count; i++ {
		if ts.eofInBuf {
			break
		}

		tok := <-ts.tokCh
		if tok.Type == eofToken {
			ts.eofInBuf = true
		}
		ts.buf = append(ts.buf, tok)
	}
}

func (ts *tokenStream) Close() error {
	if ts.eofInBuf {
		return nil
	}

	for range ts.tokCh {
		// Empty the channel
	}

	return nil
}

// Peek at the k-th token in the stream, without removing it.
// If k=0, return the next token in the stream. If k=1, return the next 2 tokens. Etc etc.
// If less than (k+1) elements are returned, the stream is at EOF
// TODO: make a diagram
func (ts *tokenStream) Peek(k int) (tokens []token) {
	// attempt to fill the buffer
	if k > len(ts.buf) {
		ts.fill(k - len(ts.buf))
	}

	// peek elements
	j := k
	if j > len(ts.buf) {
		j = len(ts.buf)
	}
	tokens = ts.buf[:j]

	return
}

func (ts *tokenStream) Next(k int) (tokens []token) {
	tokens = ts.Peek(k)
	if len(tokens) > 0 {
		ts.buf = ts.buf[len(tokens):]
	}
	return
}
