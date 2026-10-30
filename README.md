# 99-OTPs — Go Edition 🚀

> "Because apparently, passwords alone weren't stressful enough. And Python needed some competition."

Remember 99-OTPs?

That little service that makes your users abandon whatever they're doing, open their SMS app, hunt down a six-digit code, switch back to your application, and pray they haven't exceeded the two-minute expiration window?

Yeah. That one.

Well, guess who's back?

**99-OTPs has entered its Go era.**

We've taken the original OTP service, introduced it to Go, gave it ConnectRPC, and told it to get serious about life.

Why?

Because apparently, sending six-digit codes wasn't exciting enough. We needed goroutines involved.

Welcome to the second coming.

---

## What's in the box? 

### Go + ConnectRPC

For the people who looked at the original REST API and thought:

*"Interesting. But what if I wanted my application to communicate using RPC?"*

Well, congratulations. Your unnecessarily specific prayers have been answered.

This implementation is written in **Go** and powered by **ConnectRPC**.

Why ConnectRPC?

Because apparently, choosing between REST and gRPC wasn't enough of a personality test.

With ConnectRPC, we get:
- Strongly typed service definitions.
- Efficient RPC communication.
- Support for Connect, gRPC, and gRPC-Web protocols.
- A developer experience that doesn't make you question your career choices.

And Go?

Goroutines. Efficient concurrency. Compiled binaries.

Basically, this implementation has been hitting the gym.

---

## What happened to the Python version? 

Relax. Nobody got fired.

The original Python REST API is still alive and kicking.

We just decided that one implementation wasn't enough to satisfy our ambitions.

Here's the family:

| Edition | Technology | Communication |
|---|---|---|
| The Original | Python + FastAPI | REST API |
| The Second Coming | Go + ConnectRPC | RPC |

One project. Two implementations. Twice the opportunities to accidentally misconfigure Redis.

You're welcome.

**Still haven't met the original?**

[Meet the Python REST API](https://github.com/isaac-kwarteng/99-OTPs)

---

## How it works

The process hasn't changed. Your users' suffering remains consistent.

1. User attempts to log in.
2. Your application calls the OTP service.
3. The service generates a six-digit code.
4. An SMS provider delivers it.
5. User opens their SMS app.
6. User reads the code.
7. User switches back to your application.
8. User enters the wrong code.
9. User requests another OTP.
10. Everyone starts questioning technology.

Eventually, authentication happens.

A beautiful success story.

---

## Why Go?

Because we can.

Okay, there's slightly more to it than that.

- **Goroutines:** Handle concurrent operations without manually managing a small army of threads.
- **Performance:** Go is compiled and designed for efficient execution.
- **Type Safety:** Catch problems before your users discover them in production. Ideally.
- **Simplicity:** Straightforward code without unnecessary architectural gymnastics.
- **ConnectRPC:** Modern RPC communication with excellent Go integration.

We wanted to explore a Go implementation of 99-OTPs.

So we did.

Sometimes the best engineering decisions start with questionable curiosity.

---

## Bring Your Own SMS Provider 📱

The service is designed to work with SMS delivery integrations.

The idea is simple: you're not supposed to rebuild your entire authentication infrastructure just because you changed SMS providers.

Twilio? Africa's Talking? myCSMS?

Whatever your integration supports, the goal is flexibility.

Your OTP service should generate and verify codes. Your SMS provider should deliver them.

Everyone should stick to their job.

Unlike some people.

---

## Documentation 📚

Because apparently, developers expect to know how software works before using it.

Fair enough.

Documentation is provided for the Go implementation to help you understand its setup, configuration, and integration.

We believe in making developers' lives easier.

We also believe in reading documentation before opening an issue.

Both can be true.

---

## The Roadmap 🔭

99-OTPs is an evolving project.

The bigger picture is to provide developers with flexible OTP infrastructure across different technology stacks.

More integrations, improvements, and possibilities.

Hopefully, fewer users requesting OTPs seventeen times because they can't find their SMS app.

Although that's probably beyond our control.

---

## License

MIT.

Use it. Modify it. Build something impressive with it.

Just don't pretend you wrote the whole thing.

Isaac has feelings too.

---

## Contributing 

Found a bug? Open an issue.

Fixed a bug? Open a PR.

Have an idea? Let's discuss it.

Introduced a bug while fixing another bug?

Congratulations. You've joined the software development industry.

Open a PR.

---

## Final Words

The original 99-OTPs walked so the Go implementation could run.

Prefer Python? We've got you.

Prefer Go? We've got you too.

Prefer writing your own OTP service from scratch?

We won't stop you. We just hope you enjoy debugging expiration timestamps at 2 AM.

**99-OTPs. Because authentication is complicated enough already.**

---

*Built with Go, ConnectRPC, and an unreasonable amount of determination.*

*Created by Isaac Nana Adu Kwarteng.*

*P.S. Your OTP still expires in two minutes. Go cannot save you from that.*