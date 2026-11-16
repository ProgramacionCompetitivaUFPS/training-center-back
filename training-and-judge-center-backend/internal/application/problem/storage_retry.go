package problem

func withStorageRetries(operation func() error) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		if err = operation(); err == nil {
			return nil
		}
	}
	return err
}
