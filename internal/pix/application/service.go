package application

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	accountdomain "bc_sandbox/internal/accounts/domain"
	"bc_sandbox/internal/pix/domain"
	"bc_sandbox/internal/pix/ports"

	"github.com/gofrs/uuid"
)

type Service struct {
	repository ports.Repository
	clock      func() time.Time
}

type CreatePixKeyCommand struct {
	AccountID string
	KeyType   string
	Key       string
}

type ProcessPixCommand struct {
	PayerAccountID string
	PixKey         string
	AmountCents    int64
}

type TransitionCommand struct {
	TransactionID string
	Status        string
	Service       string
	BankID        string
	Message       string
	Metadata      map[string]interface{}
}

func NewService(repository ports.Repository) Service {
	return Service{
		repository: repository,
		clock: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s Service) CreatePixKey(command CreatePixKeyCommand) (domain.PixKey, error) {
	accountID, err := uuid.FromString(command.AccountID)
	if err != nil {
		return domain.PixKey{}, ErrInvalidAccountIdentifier
	}

	account, err := s.repository.FindAccountByID(accountID)
	if err != nil {
		return domain.PixKey{}, err
	}
	if account.Status != accountdomain.AccountStatusActive {
		return domain.PixKey{}, ErrAccountCannotReceivePixKey
	}

	pixKey, validation := domain.NewPixKey(accountID, command.KeyType, command.Key, s.clock())
	if len(validation) > 0 {
		return domain.PixKey{}, ValidationError{Code: "validation_error", Message: "entrada invalida", Fields: validation}
	}

	if _, err := s.repository.FindPixKey(pixKey.KeyValue); err == nil {
		return domain.PixKey{}, ErrPixKeyAlreadyExists
	} else if !errors.Is(err, ErrPixKeyNotFound) {
		return domain.PixKey{}, err
	}

	if err := s.repository.CreatePixKey(&pixKey); err != nil {
		return domain.PixKey{}, fmt.Errorf("%w: %v", ErrPixKeyPersistenceConflict, err)
	}
	return pixKey, nil
}

func (s Service) LookupPixKey(key string) (domain.LookupResult, error) {
	normalizedKey := domain.NormalizeLookupKey(key)
	if normalizedKey == "" {
		return domain.LookupResult{}, ValidationError{
			Code:    "validation_error",
			Message: "entrada invalida",
			Fields:  map[string]string{"key": "chave Pix e obrigatoria"},
		}
	}
	return s.repository.LookupPixKey(normalizedKey)
}

func (s Service) ListAccountPixKeys(accountIDValue string) ([]domain.PixKey, error) {
	accountID, err := uuid.FromString(accountIDValue)
	if err != nil {
		return nil, ErrInvalidAccountIdentifier
	}
	return s.repository.ListPixKeysByAccount(accountID)
}

func (s Service) ProcessPix(command ProcessPixCommand) (domain.Transaction, error) {
	payerAccountID, err := uuid.FromString(command.PayerAccountID)
	if err != nil {
		return domain.Transaction{}, ErrInvalidAccountIdentifier
	}

	transaction, validation := domain.NewTransaction(payerAccountID, command.AmountCents, command.PixKey, s.clock())
	if len(validation) > 0 {
		return domain.Transaction{}, ValidationError{Code: "validation_error", Message: "entrada invalida", Fields: validation}
	}

	payerAccount, err := s.repository.FindAccountByID(payerAccountID)
	if err != nil {
		return domain.Transaction{}, err
	}
	payerBank, err := s.repository.FindBankByID(payerAccount.BankID)
	if err != nil {
		return domain.Transaction{}, err
	}
	if !payerBank.CanInitiateTransactions() {
		transaction.PayerBankID = payerBank.ID
		transaction.ReceiverBankID = payerBank.ID
		transaction.ReceiverAccountID = payerAccount.ID
		if createErr := s.createFailedTransaction(&transaction, ErrOriginBankCannotInitiate.Error(), payerBank.ID); createErr != nil {
			return domain.Transaction{}, createErr
		}
		return transaction, ErrOriginBankCannotInitiate
	}

	lookup, err := s.repository.LookupPixKey(transaction.PixKey)
	if err != nil {
		transaction.PayerBankID = payerBank.ID
		transaction.ReceiverBankID = payerBank.ID
		transaction.ReceiverAccountID = payerAccount.ID
		if createErr := s.createFailedTransaction(&transaction, err.Error(), payerBank.ID); createErr != nil {
			return domain.Transaction{}, createErr
		}
		return transaction, err
	}

	transaction.PayerBankID = payerBank.ID
	transaction.ReceiverBankID = lookup.BankID
	transaction.ReceiverAccountID = lookup.AccountID
	if lookup.AccountID == payerAccount.ID {
		if createErr := s.createFailedTransaction(&transaction, ErrPixSelfTransferNotAllowed.Error(), payerBank.ID); createErr != nil {
			return domain.Transaction{}, createErr
		}
		return transaction, ErrPixSelfTransferNotAllowed
	}

	if err := s.repository.CreateTransaction(&transaction); err != nil {
		return domain.Transaction{}, fmt.Errorf("%w: %v", ErrTransactionPersistenceConflict, err)
	}
	if err := s.recordTransition(&transaction, "", domain.StatusCreated, domain.ServiceOriginBank, payerBank.ID, "Pix: transacao criada", map[string]interface{}{
		"amount_cents":     transaction.AmountCents,
		"payer_account_id": payerAccount.ID,
		"pix_key":          transaction.PixKey,
	}); err != nil {
		return domain.Transaction{}, err
	}

	if !payerAccount.CanOperate() {
		return s.failTransaction(&transaction, ErrPayerAccountCannotOperate, domain.ServiceOriginBank, payerBank.ID)
	}
	if err := s.advance(&transaction, domain.StatusPayerAccountValidated, domain.ServiceOriginBank, payerBank.ID, domain.TransitionMessage(domain.StatusPayerAccountValidated), map[string]interface{}{
		"account_id": payerAccount.ID,
		"bank_id":    payerBank.ID,
	}); err != nil {
		return transaction, err
	}

	if err := s.advance(&transaction, domain.StatusPixKeyConsulted, domain.ServiceOriginBank, payerBank.ID, domain.TransitionMessage(domain.StatusPixKeyConsulted), map[string]interface{}{
		"pix_key":  transaction.PixKey,
		"key_type": lookup.KeyType,
	}); err != nil {
		return transaction, err
	}

	receiverAccount, err := s.repository.FindAccountByID(lookup.AccountID)
	if err != nil {
		return s.failTransaction(&transaction, err, domain.ServiceOriginBank, payerBank.ID)
	}
	if receiverAccount.Status != accountdomain.AccountStatusActive || lookup.KeyStatus != domain.KeyStatusActive {
		return s.failTransaction(&transaction, ErrReceiverAccountCannotReceive, domain.ServiceReceiverBank, lookup.BankID)
	}
	if err := s.advance(&transaction, domain.StatusReceiverAccountIdentified, domain.ServiceOriginBank, payerBank.ID, domain.TransitionMessage(domain.StatusReceiverAccountIdentified), map[string]interface{}{
		"receiver_account_id": receiverAccount.ID,
		"receiver_bank_id":    receiverAccount.BankID,
	}); err != nil {
		return transaction, err
	}

	if payerAccount.BalanceCents < transaction.AmountCents {
		return s.failTransaction(&transaction, ErrInsufficientFunds, domain.ServiceOriginBank, payerBank.ID)
	}
	if err := s.advance(&transaction, domain.StatusBalanceValidated, domain.ServiceOriginBank, payerBank.ID, domain.TransitionMessage(domain.StatusBalanceValidated), map[string]interface{}{
		"available_balance_cents": payerAccount.BalanceCents,
		"amount_cents":            transaction.AmountCents,
	}); err != nil {
		return transaction, err
	}

	payerAccount.BalanceCents -= transaction.AmountCents
	payerAccount.UpdatedAt = s.clock()
	if err := s.repository.UpdateAccountBalance(&payerAccount); err != nil {
		return transaction, fmt.Errorf("%w: %v", ErrTransactionPersistenceConflict, err)
	}
	now := s.clock()
	transaction.ReservedAt = &now
	if err := s.advance(&transaction, domain.StatusFundsReserved, domain.ServiceOriginBank, payerBank.ID, domain.TransitionMessage(domain.StatusFundsReserved), map[string]interface{}{
		"debited_account_id": payerAccount.ID,
		"reserved_cents":     transaction.AmountCents,
	}); err != nil {
		return transaction, err
	}

	if err := s.advance(&transaction, domain.StatusSentToCentralBank, domain.ServiceOriginBank, payerBank.ID, domain.TransitionMessage(domain.StatusSentToCentralBank), map[string]interface{}{
		"authority": domain.ServiceCentralBank,
	}); err != nil {
		return transaction, err
	}

	if err := s.advance(&transaction, domain.StatusSettled, domain.ServiceCentralBank, transaction.ReceiverBankID, domain.TransitionMessage(domain.StatusSettled), map[string]interface{}{
		"settlement_authority": domain.ServiceCentralBank,
	}); err != nil {
		return transaction, err
	}
	settledAt := s.clock()
	transaction.SettledAt = &settledAt
	if err := s.repository.UpdateTransaction(&transaction); err != nil {
		return transaction, fmt.Errorf("%w: %v", ErrTransactionPersistenceConflict, err)
	}

	if err := s.advance(&transaction, domain.StatusReceiverBankNotified, domain.ServiceCentralBank, transaction.ReceiverBankID, domain.TransitionMessage(domain.StatusReceiverBankNotified), map[string]interface{}{
		"receiver_bank_id": transaction.ReceiverBankID,
	}); err != nil {
		return transaction, err
	}

	receiverAccount.BalanceCents += transaction.AmountCents
	receiverAccount.UpdatedAt = s.clock()
	if err := s.repository.UpdateAccountBalance(&receiverAccount); err != nil {
		return transaction, fmt.Errorf("%w: %v", ErrTransactionPersistenceConflict, err)
	}
	creditedAt := s.clock()
	transaction.CreditedAt = &creditedAt
	if err := s.advance(&transaction, domain.StatusReceiverAccountCredited, domain.ServiceReceiverBank, transaction.ReceiverBankID, domain.TransitionMessage(domain.StatusReceiverAccountCredited), map[string]interface{}{
		"credited_account_id": receiverAccount.ID,
		"credited_cents":      transaction.AmountCents,
	}); err != nil {
		return transaction, err
	}

	if err := s.advance(&transaction, domain.StatusCompleted, domain.ServiceReceiverBank, transaction.ReceiverBankID, domain.TransitionMessage(domain.StatusCompleted), map[string]interface{}{
		"final_status": domain.StatusCompleted,
	}); err != nil {
		return transaction, err
	}

	return transaction, nil
}

func (s Service) Transition(command TransitionCommand) (domain.Transaction, error) {
	transactionID, err := uuid.FromString(command.TransactionID)
	if err != nil {
		return domain.Transaction{}, ErrInvalidTransactionIdentifier
	}
	transaction, err := s.repository.FindTransactionByID(transactionID)
	if err != nil {
		return domain.Transaction{}, err
	}
	bankID := transaction.PayerBankID
	if command.BankID != "" {
		if parsed, err := uuid.FromString(command.BankID); err == nil {
			bankID = parsed
		}
	}
	service := command.Service
	if service == "" {
		service = domain.ServiceStateMachine
	}
	message := command.Message
	if message == "" {
		message = domain.TransitionMessage(command.Status)
	}
	if command.Status == domain.StatusSettled && service != domain.ServiceCentralBank {
		_ = s.recordInvalidTransition(transaction, command.Status, service, bankID, ErrCentralBankSettlementOnly.Error(), command.Metadata)
		return transaction, ErrCentralBankSettlementOnly
	}
	if err := s.advance(&transaction, command.Status, service, bankID, message, command.Metadata); err != nil {
		return transaction, err
	}
	return transaction, nil
}

func (s Service) GetTransaction(transactionIDValue string) (domain.Transaction, error) {
	transactionID, err := uuid.FromString(transactionIDValue)
	if err != nil {
		return domain.Transaction{}, ErrInvalidTransactionIdentifier
	}
	return s.repository.FindTransactionByID(transactionID)
}

func (s Service) ListBankTransactions(bankIDValue string) ([]domain.Transaction, error) {
	bankID, err := uuid.FromString(bankIDValue)
	if err != nil {
		return nil, ErrInvalidAccountIdentifier
	}
	return s.repository.ListTransactionsByBank(bankID)
}

func (s Service) Timeline(transactionIDValue string) ([]domain.TimelineItem, error) {
	transactionID, err := uuid.FromString(transactionIDValue)
	if err != nil {
		return nil, ErrInvalidTransactionIdentifier
	}
	events, err := s.repository.ListTransactionEvents(transactionID)
	if err != nil {
		return nil, err
	}
	timeline := make([]domain.TimelineItem, 0, len(events))
	for _, event := range events {
		timeline = append(timeline, domain.TimelineItem{
			Time:        event.CreatedAt,
			Status:      event.Next,
			Description: event.Message,
			Service:     event.Service,
			BankID:      event.BankID,
			Metadata:    event.Metadata,
		})
	}
	return timeline, nil
}

func (s Service) createFailedTransaction(transaction *domain.Transaction, reason string, bankID uuid.UUID) error {
	transaction.Status = domain.StatusFailed
	transaction.FailureReason = reason
	if transaction.PayerBankID == uuid.Nil {
		transaction.PayerBankID = bankID
	}
	if transaction.ReceiverBankID == uuid.Nil {
		transaction.ReceiverBankID = bankID
	}
	if transaction.ReceiverAccountID == uuid.Nil {
		transaction.ReceiverAccountID = transaction.PayerAccountID
	}
	if err := s.repository.CreateTransaction(transaction); err != nil {
		return fmt.Errorf("%w: %v", ErrTransactionPersistenceConflict, err)
	}
	return s.recordTransition(transaction, "", domain.StatusFailed, domain.ServiceOriginBank, bankID, reason, map[string]interface{}{"reason": reason})
}

func (s Service) failTransaction(transaction *domain.Transaction, cause error, service string, bankID uuid.UUID) (domain.Transaction, error) {
	reason := cause.Error()
	transaction.FailureReason = reason
	if err := s.advance(transaction, domain.StatusFailed, service, bankID, reason, map[string]interface{}{"reason": reason}); err != nil {
		return *transaction, err
	}
	return *transaction, cause
}

func (s Service) advance(transaction *domain.Transaction, next string, service string, bankID uuid.UUID, message string, metadata map[string]interface{}) error {
	previous := transaction.Status
	if !domain.CanTransition(previous, next) && !(previous == "" && next == domain.StatusCreated) {
		_ = s.recordInvalidTransition(*transaction, next, service, bankID, ErrInvalidTransition.Error(), metadata)
		return ErrInvalidTransition
	}
	if next == domain.StatusSettled && service != domain.ServiceCentralBank {
		_ = s.recordInvalidTransition(*transaction, next, service, bankID, ErrCentralBankSettlementOnly.Error(), metadata)
		return ErrCentralBankSettlementOnly
	}
	transaction.Status = next
	transaction.UpdatedAt = s.clock()
	if err := s.repository.UpdateTransaction(transaction); err != nil {
		return fmt.Errorf("%w: %v", ErrTransactionPersistenceConflict, err)
	}
	return s.recordTransition(transaction, previous, next, service, bankID, message, metadata)
}

func (s Service) recordTransition(transaction *domain.Transaction, previous string, next string, service string, bankID uuid.UUID, message string, metadata map[string]interface{}) error {
	event, err := s.newTransactionEvent(*transaction, domain.EventTypeTransition, previous, next, service, bankID, message, metadata)
	if err != nil {
		return err
	}
	return s.repository.CreateTransactionEvent(&event)
}

func (s Service) recordInvalidTransition(transaction domain.Transaction, next string, service string, bankID uuid.UUID, message string, metadata map[string]interface{}) error {
	if metadata == nil {
		metadata = map[string]interface{}{}
	}
	metadata["attempted_status"] = next
	event, err := s.newTransactionEvent(transaction, domain.EventTypeInvalidState, transaction.Status, transaction.Status, service, bankID, message, metadata)
	if err != nil {
		return err
	}
	return s.repository.CreateTransactionEvent(&event)
}

func (s Service) newTransactionEvent(transaction domain.Transaction, eventType string, previous string, next string, service string, bankID uuid.UUID, message string, metadata map[string]interface{}) (domain.TransactionEvent, error) {
	if metadata == nil {
		metadata = map[string]interface{}{}
	}
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return domain.TransactionEvent{}, err
	}
	if bankID == uuid.Nil {
		bankID = transaction.PayerBankID
	}
	return domain.TransactionEvent{
		ID:            uuid.Must(uuid.NewV4()),
		TransactionID: transaction.ID,
		Type:          transaction.Type,
		EventType:     eventType,
		Previous:      previous,
		Next:          next,
		Message:       message,
		Service:       service,
		BankID:        bankID,
		Metadata:      string(metadataBytes),
		CreatedAt:     s.clock(),
	}, nil
}
