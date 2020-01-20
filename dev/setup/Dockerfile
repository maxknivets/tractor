# This Dockerfile is not for use so much as to verify a working dev environment
# from scratch. Building this with `make test-env` will show missing dependencies
# and help clarify what software is necessary for a new development setup.

FROM golang:1.13.6-alpine3.10
RUN apk --update add nodejs yarn curl make g++ python bash git ca-certificates
RUN yarn global add typescript
RUN git clone https://github.com/manifold/qtalk
RUN make -C qtalk link
