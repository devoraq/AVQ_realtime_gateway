package usecases

type ChatUsecase struct {
}

func NewChatUsecase() *ChatUsecase {
	return &ChatUsecase{}
}

func (uc *ChatUsecase) SendMessage(userID string, message string) error { return nil }
