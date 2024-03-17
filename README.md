# Gitea auto-backup

This is script to backup all repositories from a Gitea server.

## Usage

**First go to `config.env` and set the following variables:**
>GITEA_HOST => The host of the Gitea server
>
>GITEA_ACCESS_TOKEN => The access token of the Gitea server. To generate access token, go to your profile in gitea, go to setting, applications, generate new token (make sure to note it down, as it will not be shown again)
>
>TARGET_DIR => The directory where the backups will be stored

Sample Information is provided in the `config.env` file, you must change the values as per your requirement.
Note: if confused, kindly write the issue, I will help you out.

Then run the following command:

```bash
    go mod tidy && go run .
```

### Filtering Repositories by Flags

The script now supports filtering repositories with the following flags:

- `--onlyme`: Backs up only the repositories owned by the user associated with the provided access token.

Example usage:

```bash
    go mod tidy && go run . --onlyme
```

- `--user`: Backs up repositories owned by a specified username. This allows for backing up repositories of a specific user.

Example usage:

```bash
    go mod tidy && go run . --user="username"
```

or

```bash
    go mod tidy && go run . --user username
```

whereas username is the username of the user whose repositories you want to backup.

## Author

[👤 **Sagar Yadav**](https://www.linkedin.com/in/sagaryadav)
