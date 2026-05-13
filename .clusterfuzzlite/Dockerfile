FROM gcr.io/oss-fuzz-base/base-builder-go

COPY . $SRC/codeql-development-toolkit
WORKDIR $SRC/codeql-development-toolkit
COPY .clusterfuzzlite/build.sh $SRC/build.sh
