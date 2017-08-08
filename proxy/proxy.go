package proxy

type AppProxy interface {
	SubmitCh() chan []byte
	CommitTx(tx []byte) error
}

type BabbleProxy interface {
	CommitCh() chan []byte
	SubmitTx(tx []byte) error
}
