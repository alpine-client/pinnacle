#!/bin/bash
set -eux
url="https://github.com/AppImage/AppImageKit/releases/download/continuous/appimagetool-x86_64.AppImage"
curl -sSL ${url} -o appimagetool-x86_64.AppImage
chmod a+x appimagetool-x86_64.AppImage
