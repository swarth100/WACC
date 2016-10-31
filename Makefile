all:
	git config --global url.git@gitlab.doc.ic.ac.uk:.insteadOf https://gitlab.doc.ic.ac.uk/
	mkdir -p src/gitlab.doc.ic.ac.uk/lab1617_autumn
	git clone git@gitlab.doc.ic.ac.uk:lab1617_autumn/wacc_34.git src/gitlab.doc.ic.ac.uk/lab1617_autumn/wacc_34 --branch wacc
	GOPATH=`pwd` PATH="$$GOPATH/bin:$$PATH" $(MAKE) -C src/gitlab.doc.ic.ac.uk/lab1617_autumn/wacc_34 install

clean:
	$(RM) -r bin pkg src
