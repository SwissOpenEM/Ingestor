package globustransfer

type Notifier interface {
	OnTransferProgress(bytesTransferred int, filesTransferred int)
	OnTransferFinished()
	OnTransferCancelled()
	OnTransferFailed(err error)
}
