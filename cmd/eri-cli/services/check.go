package services

type CheckSvc struct {
}

type CheckResult struct {
}

func (c *CheckSvc) Check(email string) CheckResult {
	return CheckResult{}
}
