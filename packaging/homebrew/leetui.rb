# Homebrew formula for leetui.
#
# This is the template. A tap is its own repository — `Nano-AI/homebrew-tap`, with this
# file at `Formula/leetui.rb` — so it cannot be created from here. What CAN be prepared is
# the formula itself, and this is it: after the first `v*` tag, fill in the four sha256
# lines from the release's checksums.txt and push it to the tap.
#
#     brew tap Nano-AI/tap
#     brew install leetui
#
# The release workflow publishes checksums.txt alongside the archives precisely so this
# has something to point at.
class Leetui < Formula
  desc "Terminal client for LeetCode — browse, solve, run, and submit"
  homepage "https://github.com/Nano-AI/leetui"
  version "0.1.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Nano-AI/leetui/releases/download/v#{version}/leetui_v#{version}_darwin_arm64.tar.gz"
      sha256 "REPLACE_WITH_darwin_arm64_SHA"
    else
      url "https://github.com/Nano-AI/leetui/releases/download/v#{version}/leetui_v#{version}_darwin_amd64.tar.gz"
      sha256 "REPLACE_WITH_darwin_amd64_SHA"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/Nano-AI/leetui/releases/download/v#{version}/leetui_v#{version}_linux_arm64.tar.gz"
      sha256 "REPLACE_WITH_linux_arm64_SHA"
    else
      url "https://github.com/Nano-AI/leetui/releases/download/v#{version}/leetui_v#{version}_linux_amd64.tar.gz"
      sha256 "REPLACE_WITH_linux_amd64_SHA"
    end
  end

  def install
    bin.install "leetui"
  end

  # No `depends_on`. leetui is one static binary: CGO is off and D-009's SQLite is a
  # pure-Go translation, so there is nothing to link against at install time.
  #
  # The language toolchains are deliberately NOT dependencies either — Python, Go, C++ and
  # Node are needed only to run solutions locally, and which of them you want is your
  # business. `leetui doctor` reports what is missing and what it costs you.

  test do
    assert_match "leetui", shell_output("#{bin}/leetui --version")
    assert_match "terminal client", shell_output("#{bin}/leetui help")
  end
end
