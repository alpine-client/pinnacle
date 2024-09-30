package zenity

// Password displays the password dialog.
//
// Valid options: Title, OKLabel, CancelLabel, ExtraButton,
// WindowIcon, Attach, Modal, Username.
//
// May return: ErrCanceled, ErrExtraButton.
func Password(options ...Option) (usr string, pwd string, err error) {
	return password(applyOptions(options))
}

// Username returns an Option to display the username.
func Username() Option {
	return funcOption(func(o *options) { o.username = true })
}
