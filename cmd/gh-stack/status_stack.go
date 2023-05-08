package main

type StatusStack struct {
	LocalStack LocalStack
}

func (s *StatusStack) Load(c Config) error {
	if err := s.LocalStack.Load(c); err != nil {
		return err
	}
	return nil
}
