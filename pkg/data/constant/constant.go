package constant

const (
	MeterProviderName    = "go-template-proj-meter"
	InvalidResHttpClient = 0
)

type responseCode string

const (
	Success                       responseCode = "00" // Approved or completed successfully
	InvalidMerchant               responseCode = "03" // Invalid merchant
	NetworkMessageInvalid         responseCode = "05" // Network Message Invalid
	InvalidTransaction            responseCode = "12" // Invalid transaction
	InvalidAmount                 responseCode = "13" // Invalid amount.
	InvalidCardNumber             responseCode = "14" // Invalid card number (no such number)
	NoSuchIssuer                  responseCode = "15" // No such issuer
	CostumerCancellation          responseCode = "17" // Customer Cancellation
	InvalidResponse               responseCode = "20" // Invalid Response
	HardwareError                 responseCode = "21" // Hardware Error
	SuspectedMalfunction          responseCode = "22" // Suspected Malfunction
	FormatErrorOrInvalidSignature responseCode = "30" // Format error/Invalid Siganature
	BankNotSupported              responseCode = "31" // Bank not supported by switch
	ReqFuncNotSupported           responseCode = "40" // Requested function not supported
	InsufficientFunds             responseCode = "51" // Insufficient funds
	TransPermittedToCardholder    responseCode = "57" // Transaction not permitted to cardholder / QR is expired
	TransNotPermittedToTerminal   responseCode = "58" // Transaction not permitted to terminal
	SuspectedFraud                responseCode = "59" // Suspected Fraud
	ExceedsAmountTransLimit       responseCode = "61" // Exceeds amount transaction limit
	RestrictedCard                responseCode = "62" // Restricted card
	ExceedsFrequencyTransLimit    responseCode = "65" // Exceeds frequency transaction limit
	ResponseLateOrTimeout         responseCode = "68" // Response received too late / timeout
	UnableToDecryptTrack2         responseCode = "69" // Unable to Decrypt Track2
	NoAccounts                    responseCode = "83" // No accounts
	LinkDown                      responseCode = "89" // Link down
	CutoffIsInProcess             responseCode = "90" // Cutoff is in process, a switch is ending business for a day and starting the next (transaction can be sent again in a few minutes)
	IssuerOrSwitchIsInoperative   responseCode = "91" // Issuer or switch is inoperative
	UnableToRouteTrans            responseCode = "92" // Unable to route transaction
	DuplicateTrans                responseCode = "94" // Duplicate transaction / QR is duplicate
	SystemMalfunction             responseCode = "96" // System malfunction / system error
	ReverseDebitCustomer          responseCode = "A0" // (A Zero) Reverse Debit Customer’s Balance QR Payment Fail (Mutation not perform)
	AdditionalDataInvalid         responseCode = "N1" // Additional Data Invalid
)
