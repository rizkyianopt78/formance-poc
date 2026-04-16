local claims = {
  email_verified: false,
} + std.extVar('claims');

{
  identity: {
    traits: {
      email: claims.email,
      name: claims.name,
      picture: claims.picture
    },
  },
}