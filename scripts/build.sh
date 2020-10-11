export GOPATH=/opt
export PROJECT_AUTH=smujaddid
export PROJECT_NAME=rancher-letsencrypt

mkdir -p $GOPATH/src/github.com/$PROJECT_AUTH/
rm -rf $GOPATH/src/github.com/$PROJECT_AUTH/$PROJECT_NAME
ln -s $GOPATH/src/$PROJECT_NAME $GOPATH/src/github.com/$PROJECT_AUTH/$PROJECT_NAME
cd $GOPATH/src/$PROJECT_NAME
make deps && \
make -j 4 build
