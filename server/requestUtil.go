package server

import "github.com/google/uuid"

type RequestInfoGenertor interface {
	GetRequestId() string
	GenerateRequestId() RequestInfoGenertor
}

type requestInfoGenertorImpl struct {
	requestId string
}

func (r requestInfoGenertorImpl) GenerateRequestId() RequestInfoGenertor {
	r = requestInfoGenertorImpl{requestId: uuid.New().String()}
	return &r
}

func (r *requestInfoGenertorImpl) GetRequestId() string {
	if len(r.requestId) == 0 {
		r.requestId = uuid.New().String()
	}
	return r.requestId
}

func CreateRequestInfoGenertor() RequestInfoGenertor {
	return &requestInfoGenertorImpl{}
}
