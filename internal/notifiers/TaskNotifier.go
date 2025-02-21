package notifiers

// Interface to notify about progress of a specific task
type TransferNotifier interface {
	OnTransferProgress(bytesTransferred int, filesTransferred int)
	OnTransferCompleted()
	OnTransferCancelled()
	OnTransferFailed(err error)
}
