#!/usr/bin/env bats

load test_helper

setup() {
  global_setup
  create_app
}

teardown() {
  destroy_app
  global_teardown
}

@test "(build-env) special characters" {
  run /bin/bash -c "dokku config:set --no-restart $TEST_APP NEWRELIC_APP_NAME='$TEST_APP (Staging)'"
  echo "output: $output"
  echo "status: $status"
  assert_success

  run deploy_app
  echo "output: $output"
  echo "status: $status"
  assert_success
  run /bin/bash -c "dokku config $TEST_APP"
  assert_success
}

@test "(build-env) default curl timeouts" {
  run /bin/bash -c "dokku config:unset --global CURL_CONNECT_TIMEOUT"
  echo "output: $output"
  echo "status: $status"
  assert_success

  run /bin/bash -c "dokku config:unset --global CURL_TIMEOUT"
  echo "output: $output"
  echo "status: $status"
  assert_success

  run deploy_app
  echo "output: $output"
  echo "status: $status"
  assert_success
  run /bin/bash -c "dokku config:get --global CURL_CONNECT_TIMEOUT | grep 90"
  echo "output: $output"
  echo "status: $status"
  assert_success

  run /bin/bash -c "dokku config:get --global CURL_TIMEOUT | grep 600"
  echo "output: $output"
  echo "status: $status"
  assert_success
}

@test "(build-env) buildpack failure" {
  run /bin/bash -c "dokku config:set --no-restart $TEST_APP BUILDPACK_URL='https://github.com/dokku/fake-buildpack'"
  echo "output: $output"
  echo "status: $status"
  assert_success

  run deploy_app
  echo "output: $output"
  echo "status: $status"
  assert_failure
}

@test "(build-env) buildpack deploy with Dockerfile" {
  run /bin/bash -c "dokku config:set --no-restart $TEST_APP BUILDPACK_URL='https://github.com/dokku/heroku-buildpack-null'"
  echo "output: $output"
  echo "status: $status"
  assert_success

  run deploy_app python dokku@$DOKKU_DOMAIN:$TEST_APP move_dockerfile_into_place
  echo "output: $output"
  echo "status: $status"
  assert_success

  run /bin/bash -c "dokku --quiet config:get $TEST_APP DOKKU_APP_TYPE"
  echo "output: $output"
  echo "status: $status"
  assert_output "herokuish"
}

@test "(build-env) DOKKU_ROOT cache bind is used by default" {
  run deploy_app
  echo "output: $output"
  echo "status: $status"
  assert_success

  BUILD_CID=$(docker ps -a | grep $TEST_APP | grep /bin/bash | awk '{print $1}' | head -n1)
  run /bin/bash -c "docker inspect --format '{{ .HostConfig.Binds }}' $BUILD_CID | tr -d '[]' | cut -f1 -d:"
  echo "output: $output"
  echo "status: $status"
  assert_output "$DOKKU_ROOT/$TEST_APP/cache"
}

@test "(build-env) DOKKU_HOST_ROOT cache bind is used if set" {
  TMP_ROOT=$(mktemp -d)
  mkdir -p $DOKKU_ROOT/.dokkurc
  echo export DOKKU_HOST_ROOT="$TMP_ROOT" >$DOKKU_ROOT/.dokkurc/HOST_ROOT
  DOKKU_HOST_ROOT="$TMP_ROOT" deploy_app

  BUILD_CID=$(docker ps -a | grep $TEST_APP | grep /bin/bash | awk '{print $1}' | head -n1)
  run /bin/bash -c "docker inspect --format '{{ .HostConfig.Binds }}' $BUILD_CID | tr -d '[]' | cut -f1 -d:"
  echo "output: $output"
  echo "status: $status"
  assert_output "$TMP_ROOT/$TEST_APP/cache"

  rm -rf $TMP_ROOT $DOKKU_ROOT/.dokkurc/HOST_ROOT
}
