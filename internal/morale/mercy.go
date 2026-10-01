package morale

// AnswerMercy is wired by the game-loop adapter. Tokens are validated there.
var AnswerMercy func(userID int, token, answer string) error
