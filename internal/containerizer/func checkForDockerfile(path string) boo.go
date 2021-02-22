func checkForDockerfile(path string) bool {
	finfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Errorf("There is no file at path %s Error: %q", path, err)
			return false
		}
		log.Errorf("There was an error accessing the file at path %s Error: %q", path, err)
		return false
	}
	if finfo.IsDir() {
		log.Errorf("The path %s points to a directory. Expected a Dockerfile.", path)
		return false
	}
	return true
}