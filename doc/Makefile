all:
	pdflatex filo.tex

bib:
	pdflatex filo.tex
	bibtex filo
	pdflatex filo.tex
	pdflatex filo.tex

clean:
	rm *.aux *.log *.dvi

distclean: clean
	rm *.bbl *.blg
