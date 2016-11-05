all:
	mkdir -p src/gitlab.doc.ic.ac.uk/lab1617_autumn/wacc_34
	cp -a .git src/gitlab.doc.ic.ac.uk/lab1617_autumn/wacc_34/
	cd src/gitlab.doc.ic.ac.uk/lab1617_autumn/wacc_34; git checkout wacc
	GOPATH=`pwd` PATH="$$GOPATH/bin:$$PATH" $(MAKE) -C src/gitlab.doc.ic.ac.uk/lab1617_autumn/wacc_34 install

clean:
	$(RM) -r bin pkg src
