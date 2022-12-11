cmake_minimum_required(VERSION 3.11)
project(Deledatio LANGUAGES C VERSION 2.0.0)

enable_testing()

#
# cli
#
add_custom_target(cli ALL
  go build -o ${CMAKE_BINARY_DIR}/cli -ldflags "-X main.version=${PROJECT_VERSION}"
  WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}/cli
)

#
# copy agent to image
#
add_custom_target(cpy ALL
  cp ${CMAKE_BINARY_DIR}/delegatio-agent ${CMAKE_SOURCE_DIR}/images/mkosi.skeleton/usr/bin/delegatio-agent
  WORKING_DIRECTORY ${CMAKE_BINARY_DIR}
)

#
# images
#
add_custom_target(images ALL
  sudo mkosi --force
  WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}/images
)

#
# delegatio-agent
#
add_custom_target(delegatio-agent ALL
  go build -o ${CMAKE_BINARY_DIR}/delegatio-agent
  WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}/core
)