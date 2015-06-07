func (self *Task) GetResult() (object *XenAPIObject, err error) {
	result := APIResult{}
	err = self.Client.APICall(&result, "task.get_result", self.Ref)
	if err != nil {
		return
	}
	switch ref := result.Value.(type) {
	case string:
		// @fixme: xapi currently sends us an xmlrpc-encoded string via xmlrpc.
		// This seems to be a bug in xapi. Remove this workaround when it's fixed
		re := regexp.MustCompile("^<value><array><data><value>([^<]*)</value>.*</data></array></value>$")
		match := re.FindStringSubmatch(ref)
		if match == nil {
			object = nil
		} else {
			object = &XenAPIObject{
				Ref:    match[1],
				Client: self.Client,
			}
		}
	case nil:
		object = nil
	default:
		err = fmt.Errorf("task.get_result: unknown value type %T (expected string or nil)", ref)
	}
	return
}
