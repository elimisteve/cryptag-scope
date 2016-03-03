# CrypTag Scope

[CrypTag](https://github.com/elimisteve/cryptag) is free, open source
software that enables users to access their data from their mobile
device (Ubuntu Phone) or desktop.  It encrypts this data and stores it
in a way that it can be queried by tag, even though it's encrypted.
CrypTag stores data in a zero-knowledge way, meaning that the server
storing your data has no idea what it's storing.

(Here are slides from my DEFCON talk explaining how this works:
https://talks.stevendphillips.com/cryptag-defcon23-cryptovillage/ .)

After installing CrypTag Scope on your Ubuntu Phone...

1. **Sign into Sandstorm Oasis** at https://oasis.sandstorm.io

2. **Install the CrypTag Sandstorm web app** from
https://apps.sandstorm.io/app/mkq3a9jyu6tqvzf7ayqwg620q95p438ajs02j0yx50w2aav4zra0?experimental=true
or
https://apps.sandstorm.io/app/mkq3a9jyu6tqvzf7ayqwg620q95p438ajs02j0yx50w2aav4zra0

3. **Create a new CrypTag folder/instance/grain** on Sandstorm

4. **Click on the key icon** at the top of your screen to create an
API key to give to CrypTag Scope and whatever other apps that will
access this data, such as `cpass-sandstorm` (see below).

5. Now on CrypTag Scope on your Ubuntu Phone, **click Settings** (the
config wheel icon in the upper right), and **input your Sandstorm API
key**.

Setup complete!  See below for further usage instructions.


## Demo

You can see a demo of the CrypTag web app that will store the
(encrypted) data you fetch from CrypTag Scope at
https://oasis.sandstorm.io/appdemo/mkq3a9jyu6tqvzf7ayqwg620q95p438ajs02j0yx50w2aav4zra0
.


## Using CrypTag Scope as a note viewer or password manager

CrypTag Scope currently lets you _view_ your passwords and notes
you've stored, but what about creating that content in the first
place?

The easiest way to do this is from a 64-bit Linux desktop using
`cpass-sandstorm`, which stores data in a CrypTag web app running on
Sandstorm.  From your Linux desktop, run:

    $ mkdir ~/bin; cd ~/bin && wget https://github.com/elimisteve/cryptag/blob/master/bin/cpass-sandstorm?raw=true -O cpass-sandstorm && chmod +x cpass-sandstorm && ./cpass-sandstorm

Now, use your Sandstorm API key mentioned earlier and run

    $ ./cpass-sandstorm init <YOUR_SANDSTORM_API_KEY_GOES_HERE>

You can now create notes and passwords, including associated tags!
The syntax is:

    $ ./cpass-sandstorm create <note_or_password> mytag1 mytag2 mytag3 ...

Examples:

    $ ./cpass-sandstorm create 'This is a note' type:note reminder test

    $ ./cpass-sandstorm create mycr4zyp4ssw0rd type:password twitter @myusername

(Note that you should add the 'type:password' tag if you want CrypTag
Scope to recognize it as a password and not other types of content.
The same goes for recognizing notes and typing 'type:note'.)

To fetch your Twitter password and automatically add the first match
to your clipboard, type

    $ ./cpass-sandstorm @myusername

where `@myusername` is simply a tag you've tagged your Twitter
password with.

You can see all the text content you've stored in Sandstorm by typing

    $ ./cpass-sandstorm all


## CrypTag Scope on Ubuntu Phone

Now that you've told CrypTag Scope where it's storing its data --
namely in Sandstorm using the API key you've given it -- you can now
click Notes or Passwords from the scope drop-down to see your content.

Note that tapping on a password in your search results does something
special: the scope does a search query _to itself_ so that you can
copy and paste your password into whichever website or app you're
signing into!  Normally text within an Ubuntu Scope cannot be copied,
as far as I know.

And when you do a search in CrypTag Scope, you're searching your
content by the tags you originally attached to your data.

Also note that _CrypTag Scope only ever sends existing tags to the
server_, never what literally occurs in the search box.  (So don't
worry about your passwords showing up in the "search" box!  They are
not sent to the server in plaintext; non-existent tags are ignored
from search queries.)


## What's Next

I continue to try to make CrypTag better by making the command line
clients more useful and powerful, and to let you store your data
wherever you want -- Sandstorm, Dropbox, your own server, a local
folder, all of which [currently
work](https://github.com/elimisteve/cryptag/tree/master/cmd).  If you
have a use case that you'd like CrypTag support, feel free to [contact
me](https://twitter.com/elimisteve) and let me know -- especially if
you're a computer geek, activist, or journalist.


### Storing Files

If you're interested in storing not just passwords and notes in
Sandstorm, but also files, let me know!  This will launch soon and
[I'd love some beta testers](https://twitter.com/elimisteve).


### Desktop App

Next up: **a desktop CrypTag app** for all Linux, Windows, and Mac OS
X!  [Let me know](https://twitter.com/elimisteve) if you'd like to be
a beta tester.
