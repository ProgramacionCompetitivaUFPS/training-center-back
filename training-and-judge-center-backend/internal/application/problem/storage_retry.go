package problem

// maxRetries is the number of attempts for a storage deletion that runs after the database has
// already changed: enough to ride out a transient failure without holding the response up.
const maxRetries = 3

func withStorageRetries(operation func() error) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		if err = operation(); err == nil {
			return nil
		}
	}
	return err
}
