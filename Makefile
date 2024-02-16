validate_version:
ifndef VERSION
	$(error VERSION is undefined)
endif

release: validate_version
	# linux
	GOOS=linux go build -ldflags "-X main.version=${VERSION}" -o iso-load ;\
	tar -zcvf ./releases/iso-load_${VERSION}_linux.tar.gz ./iso-load ;\

	# macos (arm)
	GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version=${VERSION}" -o iso-load ;\
	tar -zcvf ./releases/iso-load_${VERSION}_macos_arm64.tar.gz ./iso-load ;\

	# macos (amd)
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=${VERSION}" -o iso-load ;\
	tar -zcvf ./releases/iso-load_${VERSION}_macos_amd64.tar.gz ./iso-load ;\

	# windows
	GOOS=windows go build -ldflags "-X main.version=${VERSION}" -o iso-load ;\
	tar -zcvf ./releases/iso-load_${VERSION}_windows.tar.gz ./iso-load ;\

	rm ./iso-load