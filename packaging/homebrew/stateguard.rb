class Stateguard < Formula
  desc "State protection platform for destructive Docker Compose and Kubernetes operations"
  homepage "https://github.com/kusuridheeraj/Stateguard"
  url "https://github.com/kusuridheeraj/Stateguard/archive/refs/tags/v0.1.0-dev.tar.gz"
  version "0.1.0-dev"
  sha256 "replace-me"

  def install
    bin.install "stateguard"
  end
end
